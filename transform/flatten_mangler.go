package transform

import (
	"fmt"
	"reflect"

	"github.com/fatih/structtag"
	"github.com/vimeo/dials/tagformat/caseconversion"
)

const dialsTag = "dials"

// FlattenMangler implements the Mangler interface
type FlattenMangler struct {
	tag              string
	nameEncodeCasing caseconversion.EncodeCasingFunc
	tagEncodeCasing  caseconversion.EncodeCasingFunc
}

// DefaultFlattenMangler is the default FlattenMangler
var DefaultFlattenMangler = &FlattenMangler{
	tag:              dialsTag,
	nameEncodeCasing: caseconversion.EncodeUpperCamelCase,
	tagEncodeCasing:  caseconversion.EncodeCasePreservingSnakeCase,
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

	// get the underlying element kind and ignore the underlying type here
	k, _ := getUnderlyingKindType(sf.Type)

	out := []reflect.StructField{}

	tag, prefixTag, tagErr := f.getTag(sf.Name, nil, sf.Tag)
	if tagErr != nil {
		return out, tagErr
	}

	switch k {
	case reflect.Struct:
		var err error
		out, err = f.flattenStruct([]string{sf.Name}, prefixTag, sf)
		if err != nil {
			return out, err
		}
	default:
		flattenedName := []string{sf.Name}
		name := f.nameEncodeCasing(flattenedName)

		newsf := reflect.StructField{
			Name: name,
			Type: sf.Type,
			Tag:  tag,
		}
		out = []reflect.StructField{newsf}
	}

	return out, nil
}

func (f *FlattenMangler) flattenStruct(fieldPrefix, tagPrefix []string, sf reflect.StructField) ([]reflect.StructField, error) {

	// get underlying type after removing pointers. Ignoring the kind
	_, ft := getUnderlyingKindType(sf.Type)

	out := make([]reflect.StructField, 0, ft.NumField())

	for i := 0; i < ft.NumField(); i++ {
		nestedsf := ft.Field(i)

		// add the current member name to the list of nested names needed for flattening
		flattenedNames := make([]string, len(fieldPrefix), len(fieldPrefix)+1)
		copy(flattenedNames, fieldPrefix)
		flattenedNames = append(flattenedNames, nestedsf.Name)

		// get the tag for the current field name
		flattenedTags := make([]string, len(tagPrefix), len(tagPrefix)+1)
		copy(flattenedTags, tagPrefix)
		tag, newFlattenedTags, tagErr := f.getTag(nestedsf.Name, flattenedTags, nestedsf.Tag)
		if tagErr != nil {
			return out, tagErr
		}

		// get the underlying type after removing pointer for each member
		// of the struct. Ignoring type
		nestedK, _ := getUnderlyingKindType(nestedsf.Type)
		switch nestedK {
		case reflect.Struct:
			flattened, err := f.flattenStruct(flattenedNames, newFlattenedTags, nestedsf)
			if err != nil {
				return out, err
			}
			out = append(out, flattened...)
		default:
			name := f.nameEncodeCasing(flattenedNames)
			newSF := reflect.StructField{
				Name: name,
				Type: nestedsf.Type,
				Tag:  tag,
			}
			out = append(out, newSF)
		}
	}

	return out, nil
}

// getTag uses the tag if one already exist or creates one based on the
// configured EncodingCasing function and fieldName. It returns the new parsed
// StructTag, the updated slice of tags, and any error encountered
func (f *FlattenMangler) getTag(fieldName string, tags []string, st reflect.StructTag) (reflect.StructTag, []string, error) {
	tag, ok := st.Lookup(f.tag)

	// tag already exists so use the existing tag and append to prefix tags
	if ok {
		tags = append(tags, tag)
	} else {
		// tag doesn't already exist so use the field name
		tags = append(tags, fieldName)
	}

	tagVal := f.tagEncodeCasing(tags)

	parsedTag, parseErr := structtag.Parse(string(st))
	if parseErr != nil {
		return st, tags, parseErr
	}

	parsedTag.Set(&structtag.Tag{
		Key:     f.tag,
		Name:    tagVal,
		Options: []string{},
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

func populateStruct(originalVal reflect.Value, vs []FieldValueTuple, inputIndex int) (int, error) {

	if !originalVal.CanSet() {
		return inputIndex, fmt.Errorf("Error unmangling %s. Need addressable type, actual %q", originalVal, originalVal.Type().Kind())
	}

	kind, vt := getUnderlyingKindType(originalVal.Type())

	switch kind {
	case reflect.Struct:
		// the originalVal is a pointer and to go through the fields, we need
		// the concrete type so create a new struct and remove the pointer
		setVal := reflect.New(vt)
		val := setVal.Elem()

		// go through each member in the struct and populate. Recurse if one of
		// the members is a nested struct. Otherwise populate the field
		for i := 0; i < val.NumField(); i++ {
			nestedVal := val.Field(i)
			// remove pointers to get the underlying kind. Ignoring the type
			kind, _ := getUnderlyingKindType(nestedVal.Type())

			switch kind {
			case reflect.Struct:
				var err error
				inputIndex, err = populateStruct(nestedVal, vs, inputIndex)
				if err != nil {
					return inputIndex, err
				}
			default:
				if !nestedVal.CanSet() {
					return inputIndex, fmt.Errorf("Nested value %s under %s cannot be set", nestedVal, originalVal)
				}

				if !vs[inputIndex].Value.Type().AssignableTo(nestedVal.Type()) {
					return inputIndex, fmt.Errorf("Error unmangling. Expected type %s. Actual type %s", vs[inputIndex].Value.Type(), nestedVal.Type())
				}
				nestedVal.Set(vs[inputIndex].Value)
				inputIndex++
			}
		}
		setVal.Elem().Set(val)
		originalVal.Set(setVal)
	default:
		originalVal.Set(vs[inputIndex].Value)
		inputIndex++
	}

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
