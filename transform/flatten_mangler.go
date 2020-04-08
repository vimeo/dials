package transform

import (
	"fmt"
	"reflect"
)

// FlattenMangler implements the Mangler interface
type FlattenMangler struct{}

// Mangle goes through each StructField and flattens the structure
func (f *FlattenMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	// Make sure we're pointerized (or nilable). Should have called pointerify before
	// calling this function
	switch sf.Type.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
	default:
		return []reflect.StructField{}, fmt.Errorf("flag: programmer error: expected pointerized fields, got %s",
			sf.Type)
	}

	// if the StructField is a pointer, get the underlying element type
	ft := sf.Type
	k := ft.Kind()
	for k == reflect.Ptr {
		ft = ft.Elem()
		k = ft.Kind()
	}

	out := []reflect.StructField{}

	switch k {
	case reflect.Struct:
		flattenedStruct := flattenStruct(sf.Name, sf)
		out = append(out, flattenedStruct...)
	default:
		newsf := reflect.StructField{
			Name:    sf.Name,
			Type:    sf.Type,
			Tag:     sf.Tag,
			PkgPath: sf.PkgPath,
		}
		out = append(out, newsf)
	}

	for _, o := range out {
		fmt.Println("name", o.Name)
	}
	return out, nil
}

func flattenStruct(prefix string, sf reflect.StructField) []reflect.StructField {

	// add the outer struct before adding the flattened struct fields
	newSF := reflect.StructField{
		Name:    prefix,
		Type:    sf.Type,
		Tag:     sf.Tag,
		PkgPath: sf.PkgPath,
	}

	ft := sf.Type
	k := ft.Kind()
	for k == reflect.Ptr {
		ft = ft.Elem()
		k = ft.Kind()
	}

	out := []reflect.StructField{newSF}

	for i := 0; i < ft.NumField(); i++ {
		// get the underlying type if it's a pointer
		nestedsf := ft.Field(i)
		nestedt := nestedsf.Type
		nestedK := nestedt.Kind()
		for nestedK == reflect.Ptr {
			nestedt = nestedt.Elem()
			nestedK = nestedt.Kind()
		}
		name := prefix + "_" + nestedsf.Name
		switch nestedK {
		case reflect.Struct:
			flattened := flattenStruct(name, nestedsf)
			out = append(out, flattened...)
		default:
			newSF := reflect.StructField{
				Name:    name,
				Type:    sf.Type,
				Tag:     sf.Tag,
				PkgPath: sf.PkgPath,
			}
			out = append(out, newSF)
		}
	}

	return out

}

// Unmangle ...
func (f *FlattenMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {
	return reflect.Value{}, nil
}

// ShouldRecurse always returns true in order to walk nested structs.
func (f *FlattenMangler) ShouldRecurse(reflect.StructField) bool {
	return false
}
