package transform

import (
	"reflect"
)

// AnonymousFlattenMangler hoists the fields from the types of anonymous
// struct-fields into the parent type. (working around decoders/sources that
// are unaware of anonymous fields)
// Note: this mangler is unaware of TextUnmarshaler implementations (it's tricky to do right when flattening).
// It should be combined with the TextUnmarshalerMangler if the prefered
// handling is to mask the other fields in that struct with the TextUnmarshaler
// implementation.
type AnonymousFlattenMangler struct{}

// Mangle is called for every field in a struct, and maps that to one or more output fields.
// Implementations that desire to leave fields unchanged should return
// the argument unchanged. (particularly useful if taking advantage of
// recursive evaluation)
func (a AnonymousFlattenMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	// If it's not an anonymous field, return it as-is
	if !sf.Anonymous {
		return []reflect.StructField{sf}, nil
	}
	// Note: TranslateType already skips unexported fields

	// anonymous/embedded fields can only be interfaces, pointers and structs
	switch sf.Type.Kind() {
	case reflect.Pointer:
		// recurse with the pointer stripped off
		sfInner := sf
		sfInner.Type = sf.Type.Elem()
		return a.Mangle(sfInner)
	case reflect.Struct:
		out := make([]reflect.StructField, 0, sf.Type.NumField())
		for i := 0; i < sf.Type.NumField(); i++ {
			innerField := sf.Type.Field(i)
			if !innerField.IsExported() {
				// skip unexported fields
				continue
			}
			out = append(out, innerField)
		}

		return out, nil
	default:
		// leave everything else alone (there's nothing to promote)
		// this includes interfaces and all other non-struct and
		// non-pointer-to-struct types.
		return []reflect.StructField{sf}, nil
	}
}

// bool return value indicates whether all fields are nil (and as such, a nil value should be returned for pointer-types)
func (a AnonymousFlattenMangler) unmangleStruct(sf reflect.StructField, fvs []FieldValueTuple) (reflect.Value, bool) {
	out := reflect.New(sf.Type).Elem()
	if len(fvs) == 0 {
		// no fields made it, just return out.
		return out, true
	}
	fvsIdx := 0
	allNil := true
	for i := 0; i < sf.Type.NumField(); i++ {
		oft := sf.Type.Field(i)
		if oft.Name == fvs[fvsIdx].Field.Name {
			out.Field(i).Set(fvs[fvsIdx].Value)
			switch fvs[fvsIdx].Value.Kind() {
			// check for nil-able types
			case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan:
				if !fvs[fvsIdx].Value.IsZero() {
					allNil = false
				}
			default:
				// non-nilable field, just assume it's non-nil
				// pointerification shold have made this nilable, though.
				allNil = false
			}
			fvsIdx++
		}
	}
	return out, allNil
}

// Unmangle is called for every source-field->mangled-field
// mapping-set, with the mangled-field and its populated value set. The
// implementation of Unmangle should return a reflect.Value that will
// be used for the next mangler or final struct value)
// Returned reflect.Value should be convertible to the field's type.
func (a AnonymousFlattenMangler) Unmangle(sf reflect.StructField, fvs []FieldValueTuple) (reflect.Value, error) {
	if !sf.Anonymous {
		// not anonymous, just forward the single value
		return fvs[0].Value, nil
	}
	switch sf.Type.Kind() {
	case reflect.Pointer:
		// It's a pointer. check for nil; strip off the pointer and recurse
		msf := sf
		msf.Type = sf.Type.Elem()
		v, allNil := a.unmangleStruct(msf, fvs)
		if allNil {
			return reflect.Zero(sf.Type), nil
		}
		return v.Addr(), nil
	case reflect.Struct:
		out, _ := a.unmangleStruct(sf, fvs)
		return out, nil
	default:
		// not a struct-typed anonymous field, just forward up the chain
		return fvs[0].Value, nil
	}
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
func (a AnonymousFlattenMangler) ShouldRecurse(_ reflect.StructField) bool {
	return true
}
