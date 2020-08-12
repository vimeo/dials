package transform

import (
	"fmt"
	"go/ast"
	"reflect"
)

// FieldValueTuple ties together the StructField and the value to be converted
// back to the input-type
type FieldValueTuple struct {
	Field reflect.StructField
	Value reflect.Value
}

// Mangler implementations operate on a field-by-field basis
type Mangler interface {
	// Mangle is called for every field in a struct, and maps that to one or more output fields.
	// Implementations that desire to leave fields unchanged should return
	// the argument unchanged. (particularly useful if taking advantage of
	// recursive evaluation)
	Mangle(reflect.StructField) ([]reflect.StructField, error)
	// Unmangle is called for every source-field->mangled-field
	// mapping-set, with the mangled-field and its populated value set. The
	// implementation of Unmangle should return a reflect.Value that will
	// be used for the next mangler or final struct value)
	Unmangle(reflect.StructField, []FieldValueTuple) (reflect.Value, error)
	// ShouldRecurse is called after Mangle for each field so nested struct
	// fields get iterated over after any transformation done by Mangle().
	ShouldRecurse(reflect.StructField) bool
}

type fieldTransformPair struct {
	field reflect.StructField
	// If this field is a struct-type (or pointer-to-struct) and the
	// associated Mangler requested recursion on the original field, we
	// record the Transformer used for that recursive translation here.
	transform *Transformer
}

func initFieldTransformPairs(fields []reflect.StructField) []fieldTransformPair {
	out := make([]fieldTransformPair, len(fields))
	for i, f := range fields {
		out[i] = fieldTransformPair{field: f}
	}
	return out
}

type transformMappingElement struct {
	in  reflect.StructField
	out []fieldTransformPair
}

// NewTransformer constructs a Transformer instance with the specified manglers
// and type (the order of manglers specified here is the order they'll be
// evaluated in Mangle()).
func NewTransformer(t reflect.Type, manglers ...Mangler) *Transformer {
	return &Transformer{
		t:        t,
		manglers: manglers,
	}
}

// Transformer wraps a type and an arbitrary set of Manglers.
type Transformer struct {
	manglers []Mangler
	// This is a double-slice to cover two dimensions:
	//  - a dimension for manglers (outer)
	//  - a dimension for fields in the original struct (inner)
	mState [][]transformMappingElement
	t      reflect.Type
}

func unpackFields(t reflect.Type) []reflect.StructField {
	switch t.Kind() {
	case reflect.Struct:
	default:
		return nil
	}
	out := make([]reflect.StructField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out[i] = t.Field(i)
	}
	return out
}

func unpackValueFields(v reflect.Value) []FieldValueTuple {
	t := v.Type()
	switch t.Kind() {
	case reflect.Struct:
	default:
		return nil
	}
	out := make([]FieldValueTuple, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out[i] = FieldValueTuple{
			Field: t.Field(i),
			Value: v.Field(i),
		}
	}
	return out
}

func isStructTypedField(field reflect.StructField) bool {
	switch field.Type.Kind() {
	case reflect.Struct:
		return true
	case reflect.Ptr:
		if field.Type.Elem().Kind() == reflect.Struct {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *Transformer) maybeRecursivelyMangle(mangler Mangler, state *transformMappingElement,
	fields []reflect.StructField) ([]reflect.StructField, error) {
	// copy fields into another equal-length-slice
	out := append([]reflect.StructField{}, fields...)
	for i, field := range fields {
		if !isStructTypedField(field) {
			continue
		}
		if !mangler.ShouldRecurse(field) {
			continue
		}
		ft := field.Type
		// strip any outer pointerification
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		fieldTransformer := Transformer{
			manglers: []Mangler{mangler},
			mState:   nil,
			t:        ft,
		}
		state.out[i].transform = &fieldTransformer
		mangledType, manglingErr := fieldTransformer.TranslateType()
		if manglingErr != nil {
			return nil, fmt.Errorf("failed to mangle field %d (name %s): %s",
				i, field.Name, manglingErr)
		}
		// Reinstate pointerification
		if field.Type.Kind() == reflect.Ptr {
			mangledType = reflect.PtrTo(mangledType)
		}
		out[i].Type = mangledType
	}

	return out, nil
}

func clearFieldIdxs(sf []reflect.StructField) {
	for i := range sf {
		sf[i].Index = nil
	}
}

// TranslateType calls `Mangle` on all `Manglers` in order, tracking the conversion
// for use in ReverseTranslate.
func (t *Transformer) TranslateType() (reflect.Type, error) {
	// iterate through manglers in order saving the input structfield and output
	// slice of struct fields in the mState map
	layerFields := unpackFields(t.t)
	t.mState = make([][]transformMappingElement, len(t.manglers))
	for manglerNum, mangler := range t.manglers {
		manglerFields := make([]reflect.StructField, 0, len(layerFields))
		layerState := make([]transformMappingElement, len(layerFields))
		for i, structField := range layerFields {

			// Skip unexported fields
			if !ast.IsExported(structField.Name) {
				continue
			}

			fields, mangleErr := mangler.Mangle(structField)
			if mangleErr != nil {
				return nil,
					fmt.Errorf("failed to mangle field %d with mangler %d (type %T): %s",
						i, manglerNum, mangler, mangleErr)
			}
			state := transformMappingElement{
				in:  structField,
				out: initFieldTransformPairs(fields),
			}

			// Clear the field-offsets so latter unmangling code
			// can use it for figuring out whether we're going back
			// to the original type (and address the correct field
			// in that original type while we're at it).
			clearFieldIdxs(fields)
			nextFields, recurseErr := t.maybeRecursivelyMangle(mangler, &state, fields)
			if recurseErr != nil {
				return nil,
					fmt.Errorf("failed to recursively mangle field %d with mangler %d (type %T): %s",
						i, manglerNum, mangler, mangleErr)
			}

			manglerFields = append(manglerFields, nextFields...)

			layerState[i] = state
		}

		layerFields = manglerFields

		t.mState[manglerNum] = layerState
	}
	return reflect.StructOf(layerFields), nil
}

// Translate calls `TranslateType` and returns an instance of the new type (or an error)
func (t *Transformer) Translate() (reflect.Value, error) {
	outType, typeMangleErr := t.TranslateType()
	if typeMangleErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to convert type: %s", typeMangleErr)
	}

	return reflect.New(outType).Elem(), nil

}

// returns the field values after any unmangling of the constituent types of sub-fields
func (t *Transformer) maybeRecursivelyUnmangle(
	fieldState *transformMappingElement, mangledField []FieldValueTuple) ([]FieldValueTuple, *UnmangleError) {

	mf := append([]FieldValueTuple{}, mangledField...)
	for z, field := range mangledField {
		fieldTransformer := fieldState.out[z].transform
		if fieldTransformer == nil {
			continue
		}
		v := field.Value
		isPtr := false
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				// if it's nil, skip it, the unmangler isn't
				// going to do anything useful on the field of
				// a struct pointed to by a nil-pointer.
				continue
			}
			v = v.Elem()
			isPtr = true
		}
		unmangledVal, unmangleErr := fieldTransformer.ReverseTranslate(v)
		if unmangleErr != nil {
			return nil, &UnmangleError{Err: unmangleErr, ErrString: fmt.Sprintf("failed to recursively inverse transform field %s: %s",
				field.Field.Name, unmangleErr)}
		}
		mf[z].Value = unmangledVal
		if isPtr {
			mf[z].Value = unmangledVal.Addr()
		}
	}

	return mf, nil
}

// returns the field value
func (t *Transformer) unmangleField(
	manglerIdx int, fieldState *transformMappingElement, mangledField []FieldValueTuple) (reflect.Value, error) {
	// Since we recurse into field types after we've mangled the
	// field itself, we have to recurse first here.
	mf, recursivelyUnmangleErr := t.maybeRecursivelyUnmangle(fieldState, mangledField)
	if recursivelyUnmangleErr != nil {
		return reflect.Value{}, recursivelyUnmangleErr
	}

	mangler := t.manglers[manglerIdx]
	unmangledVal, unmangleErr := mangler.Unmangle(fieldState.in, mf)
	if unmangleErr != nil {
		errString := fmt.Sprintf("unmangle from mangler %d (type %T) failed: %s",
			manglerIdx, mangler, unmangleErr)

		return reflect.Value{}, &UnmangleError{Err: unmangleErr, ErrString: errString}
	}
	return unmangledVal, nil
}

// UnmangleError represents an error in unmangling.
type UnmangleError struct {
	Err       error
	ErrString string
}

// Error implements the Error interface.
func (e *UnmangleError) Error() string {
	return e.ErrString
}

// Unwrap returns in the inner error.
func (e *UnmangleError) Unwrap() error {
	return e.Err
}

// ReverseTranslateError represents an error in reverse translation.
type ReverseTranslateError struct {
	Err       error
	ErrString string
}

// Error implements the Error interface.
func (e *ReverseTranslateError) Error() string {
	return e.ErrString
}

// Unwrap returns the inner error.
func (e *ReverseTranslateError) Unwrap() error {
	return e.Err
}

// ReverseTranslate calls each Mangler's Unmangle method in reverse order.
func (t *Transformer) ReverseTranslate(v reflect.Value) (reflect.Value, error) {
	// iterate through manglers in reverse order passing the value of the struct
	// field paired with its reflect.StructField as a FieldValueTuple

	layerMangledVal := unpackValueFields(v)
	// we're iterating backwards through manglers
	for manglerNum := len(t.manglers) - 1; manglerNum >= 0; manglerNum-- {
		mangledfieldOffset := 0
		unmangledLayerVals := make([]FieldValueTuple, len(t.mState[manglerNum]))
		for srcFieldIdx, srcFieldstate := range t.mState[manglerNum] {
			// slice down to just the mangled fields we're
			// interested in for this unmangled field.
			fvtuples := layerMangledVal[mangledfieldOffset : mangledfieldOffset+len(srcFieldstate.out)]

			nextLayerVal, unmangleErr := t.unmangleField(
				manglerNum, &srcFieldstate, fvtuples)
			if unmangleErr != nil {
				errString := fmt.Sprintf("failed to unmangle field %d (%q) with mangler %d (type %T): %s",
					srcFieldIdx, srcFieldstate.in.Name, manglerNum,
					t.manglers[manglerNum], unmangleErr)

				return reflect.Value{}, &ReverseTranslateError{Err: unmangleErr, ErrString: errString}
			}

			// set the unmangled value on our field.
			unmangledLayerVals[srcFieldIdx] = FieldValueTuple{
				Value: nextLayerVal,
				Field: srcFieldstate.in,
			}

			mangledfieldOffset += len(srcFieldstate.out)
		}
		layerMangledVal = unmangledLayerVals
	}

	// Now that we've gone through all the manglers, we can reassemble the original struct value.
	outVal := reflect.New(t.t).Elem()
	for _, field := range layerMangledVal {
		if !ast.IsExported(field.Field.Name) {
			// skip unexported fields
			continue
		}
		// We preserved the indices on the original outer-struct, and
		// the other ones were cleared, so this should be safe if we
		// managed our fields properly.
		outField := outVal.FieldByIndex(field.Field.Index)
		if !field.Value.Type().ConvertibleTo(outField.Type()) {
			errString := fmt.Sprintf("incompatible types for field %q; original field type %s; final unmangled type %s",
				field.Field.Name, outField.Type(), field.Value.Type())

			return reflect.Value{}, &ReverseTranslateError{ErrString: errString}
		}
		convertedVal := field.Value.Convert(outField.Type())
		outField.Set(convertedVal)
	}
	return outVal, nil
}
