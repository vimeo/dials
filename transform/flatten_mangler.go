package transform

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/structtag"
	"github.com/vimeo/dials/tagformat/caseconversion"
)

const (
	// DialsTagName is the name of the dials tag.
	DialsTagName = "dials"
	// dialsFieldPathTag contains the path to the nested struct field as a
	// comma separated string of the nested field index.
	dialsFieldPathTag = "dialsfieldpath"
)

// textMReflectType is a reflect.Type of TextUnmarshaler
var textMReflectType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// FlattenMangler implements the Mangler interface
type FlattenMangler struct {
	tag              string
	nameEncodeCasing caseconversion.EncodeCasingFunc
	tagEncodeCasing  caseconversion.EncodeCasingFunc
}

// DefaultFlattenMangler returns a FlattenMangler with preset values for tag,
// nameEncodeCasing, and tagEncodeCasing
func DefaultFlattenMangler() *FlattenMangler {
	return &FlattenMangler{
		tag:              DialsTagName,
		nameEncodeCasing: caseconversion.EncodeUpperCamelCase,
		tagEncodeCasing:  caseconversion.EncodeCasePreservingSnakeCase,
	}
}

// NewFlattenMangler is the constructor for FlattenMangler
func NewFlattenMangler(tag string, nameEnc, tagEnc caseconversion.EncodeCasingFunc) *FlattenMangler {
	return &FlattenMangler{
		tag:              tag,
		nameEncodeCasing: nameEnc,
		tagEncodeCasing:  tagEnc,
	}
}

// Mangle goes through each StructField and flattens the structure
func (f *FlattenMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	// Make sure we're pointerized (or nilable). Should have called pointerify
	// before calling this function
	switch sf.Type.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
	default:
		return []reflect.StructField{}, fmt.Errorf("FlattenMangler: programmer error: expected pointerized fields, got %s",
			sf.Type)
	}

	// get the underlying element kind and type
	k, t := getUnderlyingKindType(sf.Type)

	out := []reflect.StructField{}
	fieldPath := []string{sf.Name}

	tag, prefixTag, tagErr := f.getTag(&sf, nil, fieldPath)
	if tagErr != nil {
		return out, tagErr
	}

	switch k {
	case reflect.Struct:
		// only flatten the struct if it doesn't implement TextUnmarshaler
		if t.Implements(textMReflectType) || reflect.PtrTo(t).Implements(textMReflectType) {
			break
		}
		fieldPrefix := []string{}
		if !sf.Anonymous {
			fieldPrefix = append(fieldPrefix, sf.Name)
		}
		return f.flattenStruct(fieldPrefix, prefixTag, fieldPath, sf)
	default:
	}

	flattenedName := []string{sf.Name}
	name := f.nameEncodeCasing(flattenedName)

	newsf := reflect.StructField{
		Name:      name,
		Type:      sf.Type,
		Tag:       tag,
		Anonymous: sf.Anonymous,
	}
	out = []reflect.StructField{newsf}

	return out, nil
}

// flattenStruct takes a struct and flattens all the fields and makes a recursive
// call if the field is a struct too
func (f *FlattenMangler) flattenStruct(fieldPrefix, tagPrefix, fieldPath []string, sf reflect.StructField) ([]reflect.StructField, error) {

	// get underlying type after removing pointers. Ignoring the kind
	_, ft := getUnderlyingKindType(sf.Type)

	out := make([]reflect.StructField, 0, ft.NumField())

	for i := 0; i < ft.NumField(); i++ {
		nestedsf := ft.Field(i)

		flattenedNames := fieldPrefix

		// add the current member name to the list of nested names needed for
		// flattening if not an embedded field
		if !nestedsf.Anonymous {
			flattenedNames = append(fieldPrefix[:len(fieldPrefix):len(fieldPrefix)], nestedsf.Name)
		}

		// Need a separate flattenPath slice for the field path instead of just
		// using the fieldPrefix one because we need to add the names of the
		// embedded fields to the slice so we can iterate through and get the original field
		flattenedPath := append(fieldPath[:len(fieldPath):len(fieldPath)], nestedsf.Name)

		// add the tag of the current field to the list of flattened tags
		tag, flattenedTags, tagErr := f.getTag(&nestedsf, tagPrefix, flattenedPath)
		if tagErr != nil {
			return out, tagErr
		}

		// get the underlying type after removing pointer for each member
		// of the struct. Ignoring type
		nestedK, nestedT := getUnderlyingKindType(nestedsf.Type)
		switch nestedK {
		case reflect.Struct:
			// don't flatten if struct implements TextUnmarshaler
			if nestedT.Implements(textMReflectType) || reflect.PtrTo(nestedT).Implements(textMReflectType) {
				break
			}
			flattened, err := f.flattenStruct(flattenedNames, flattenedTags, flattenedPath, nestedsf)
			if err != nil {
				return out, err
			}
			out = append(out, flattened...)
			continue
		default:

		}
		name := f.nameEncodeCasing(flattenedNames)
		newSF := reflect.StructField{
			Name:      name,
			Type:      nestedsf.Type,
			Tag:       tag,
			Anonymous: sf.Anonymous,
		}
		out = append(out, newSF)
	}

	return out, nil
}

// getTag uses the tag if one already exist or creates one based on the
// configured EncodingCasing function and fieldName. It returns the new parsed
// StructTag, the updated slice of tags, and any error encountered
func (f *FlattenMangler) getTag(sf *reflect.StructField, tags, flattenedPath []string) (reflect.StructTag, []string, error) {
	tag, ok := sf.Tag.Lookup(f.tag)

	// tag already exists so use the existing tag and append to prefix tags
	if ok {
		tags = append(tags[:len(tags):len(tags)], tag)
	} else if !sf.Anonymous {
		// tag doesn't already exist so use the field name as long as it's not
		// Anonymous (embedded field)
		tags = append(tags[:len(tags):len(tags)], sf.Name)

	}

	tagVal := f.tagEncodeCasing(tags)

	parsedTag, parseErr := structtag.Parse(string(sf.Tag))
	if parseErr != nil {
		return sf.Tag, tags, parseErr
	}

	parsedTag.Set(&structtag.Tag{
		Key:     f.tag,
		Name:    tagVal,
		Options: []string{},
	})

	parsedTag.Set(&structtag.Tag{
		Key:  dialsFieldPathTag,
		Name: strings.Join(flattenedPath, ","),
	})

	return reflect.StructTag(parsedTag.String()), tags, nil
}

// Unmangle goes through the struct and populates the values of the struct
// that come from the populated flattened struct fields
func (f *FlattenMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {

	val := reflect.New(sf.Type).Elem()
	output, err := populateStruct(val, vs, 0)
	if err != nil {
		return val, err
	}

	if output != len(vs) {
		return val, fmt.Errorf("Error unmangling %v. Number of input values %d not equal to number of struct fields that need values %d", sf, len(vs), output)
	}

	return val, nil
}

// populateStruct populates the original value with values from the flattend values
func populateStruct(originalVal reflect.Value, vs []FieldValueTuple, inputIndex int) (int, error) {
	if !originalVal.CanSet() {
		return inputIndex, fmt.Errorf("Error unmangling %s. Need addressable type, actual %q", originalVal, originalVal.Type().Kind())
	}

	kind, vt := getUnderlyingKindType(originalVal.Type())

	switch kind {
	case reflect.Struct:
		// go through each field if the struct doesn't implement TextUnmarshaler
		if vt.Implements(textMReflectType) || reflect.PtrTo(vt).Implements(textMReflectType) {
			break
		}
		// the originalVal is a pointer and to go through the fields, we need
		// the concrete type so create a new struct and remove the pointer
		setVal := reflect.New(vt)
		val := setVal.Elem()

		// go through each member in the struct and populate. Recurse if one of
		// the members is a nested struct. Otherwise populate the field
		for i := 0; i < val.NumField(); i++ {
			nestedVal := val.Field(i)
			// remove pointers to get the underlying kind. Ignoring the type
			kind, t := getUnderlyingKindType(nestedVal.Type())

			switch kind {
			case reflect.Struct:
				// don't flatten if the struct implements TextUnmarshaler
				if t.Implements(textMReflectType) || reflect.PtrTo(t).Implements(textMReflectType) {
					break // break out of the case, still stays within the for loop
				}
				var err error
				inputIndex, err = populateStruct(nestedVal, vs, inputIndex)
				if err != nil {
					return inputIndex, err
				}
				continue
			default:
			}
			if !nestedVal.CanSet() {
				return inputIndex, fmt.Errorf("Nested value %s under %s cannot be set", nestedVal, originalVal)
			}

			if !vs[inputIndex].Value.Type().AssignableTo(nestedVal.Type()) {
				return inputIndex, fmt.Errorf("Error unmangling. Expected type %s. Actual type %s", vs[inputIndex].Value.Type(), nestedVal.Type())
			}
			nestedVal.Set(vs[inputIndex].Value)
			inputIndex++
		}
		setVal.Elem().Set(val)
		originalVal.Set(setVal)
		return inputIndex, nil
	default:
	}
	originalVal.Set(vs[inputIndex].Value)
	inputIndex++

	return inputIndex, nil
}

// ShouldRecurse returns false because Mangle walks through nested structs and doesn't need Transform's recursion
func (f *FlattenMangler) ShouldRecurse(reflect.StructField) bool {
	return false
}

// getUnderlyingKindType strips the pointer from the type to determine the underlying kind
func getUnderlyingKindType(t reflect.Type) (reflect.Kind, reflect.Type) {
	k := t.Kind()
	for k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return k, t
}

func stripPtrs(val reflect.Value) reflect.Value {
	for val.IsValid() {
		switch val.Kind() {
		case reflect.Ptr, reflect.Interface:
			val = val.Elem()
		default:
			return val
		}
	}
	return val
}

// GetField should be called after calling the flatten mangler. It will look at
// the dialsfieldpath tag of the mangled StructFields (sf) set by the flatten
// mangler to get the path to the original field. It return the concrete value
// at the original field and if the original field value isn't populated, it will
// return the zero value.
func GetField(sf reflect.StructField, v reflect.Value) reflect.Value {
	fieldPath := sf.Tag.Get(dialsFieldPathTag)
	// the tag should always be set after going through the flatten mangler
	if fieldPath == "" {
		panic(fmt.Errorf("dialsfieldpath tag not set for field %s", sf.Name))
	}
	fields := strings.Split(fieldPath, ",")
	for _, fname := range fields {
		v = stripPtrs(v)
		// if the struct isn't populated, return the zero value
		if !v.IsValid() {
			// ignore the kind and get the concrete type
			_, t := getUnderlyingKindType(sf.Type)
			return reflect.New(t).Elem()
		}
		v = v.FieldByName(fname)
	}

	v = stripPtrs(v)

	// if the final value isn't populated, return the zero value
	if !v.IsValid() {
		// ignore the kind and get the concrete type
		_, t := getUnderlyingKindType(sf.Type)
		return reflect.New(t).Elem()
	}

	return v
}
