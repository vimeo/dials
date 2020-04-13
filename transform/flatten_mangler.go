package transform

import (
	"fmt"
	"reflect"
)

// FlattenMangler implements the Mangler interface
type FlattenMangler struct{}

// Mangle goes through each StructField and flattens the structure
func (f *FlattenMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	// Make sure we're pointerized (or nilable). Should have called pointerify
	// before calling this function
	switch sf.Type.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
	default:
		return []reflect.StructField{}, fmt.Errorf("FlattenMangler: programmer error: expected pointerized fields, got %s",
			sf.Type)
	}

	// get the underlying element kind and ignore the underlying type here
	k, _ := getUnderlyingKindType(sf.Type)

	out := []reflect.StructField{}

	switch k {
	case reflect.Struct:
		flattenedStruct := flattenStruct(sf.Name, sf)
		out = append(out, flattenedStruct...)
	default:
		newsf := reflect.StructField{
			Name: sf.Name,
			Type: sf.Type,
			Tag:  sf.Tag,
		}
		out = append(out, newsf)
	}

	return out, nil
}

func flattenStruct(prefix string, sf reflect.StructField) []reflect.StructField {

	// get underlying type after removing pointers. Ignoring the kind
	_, ft := getUnderlyingKindType(sf.Type)

	out := []reflect.StructField{}

	for i := 0; i < ft.NumField(); i++ {
		nestedsf := ft.Field(i)
		// get the underlying type after removing pointer for each member
		// of the struct. Ignoring type
		nestedK, _ := getUnderlyingKindType(nestedsf.Type)

		// concatenates the outerlayer names with the current member name
		name := prefix + "_" + nestedsf.Name
		switch nestedK {
		case reflect.Struct:
			flattened := flattenStruct(name, nestedsf)
			out = append(out, flattened...)
		default:
			newSF := reflect.StructField{
				Name: name,
				Type: nestedsf.Type,
				Tag:  sf.Tag,
			}
			out = append(out, newSF)
		}
	}

	return out
}

// Unmangle goes through the struct and populates the values of the struct
// that come from the populated flattened struct fields
func (f *FlattenMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {

	t := reflect.StructOf([]reflect.StructField{sf})
	val := reflect.New(t).Elem().Field(0)

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
		return inputIndex, fmt.Errorf("Error unmangling %s. Need addressable type, actual %q", originalVal.String(), originalVal.Type().Kind().String())
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
					return inputIndex, fmt.Errorf("Nested value %s under %s cannot be set", nestedVal.String(), originalVal.String())
				}

				if vs[inputIndex].Value.Type() != nestedVal.Type() {
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
