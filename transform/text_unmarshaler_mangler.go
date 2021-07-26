package transform

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/vimeo/dials/helper"
)

var (
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// TextUnmarshalerMangler changes types that implement encoding.TextUnmarshaler
// to string and uses that interface to cast back to their original type.
type TextUnmarshalerMangler struct{}

// Mangle changes the type of the provided StructField to string if that
// StructField type implements encoding.TextUnmarshaler.  Otherwise, the type is
// passed through unaltered.
func (*TextUnmarshalerMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	if sf.Type.Implements(textUnmarshalerType) || reflect.PtrTo(sf.Type).Implements(textUnmarshalerType) {
		sf.Type = strPtrType
	}
	return []reflect.StructField{sf}, nil
}

// Unmangle unmangles.
func (*TextUnmarshalerMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {
	return helper.OnImplements(sf.Type, textUnmarshalerType, vs[0].Value, func(input reflect.Value, v reflect.Value) (reflect.Value, error) {
		strPtr := input.Interface().(*string)
		fmt.Printf("input is %v\n", strPtr)
		if strPtr == nil {
			return reflect.Zero(v.Type()), nil
		}
		val := v.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(*strPtr))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v, %s", sf.Type, err)
		}
		return v, nil
	})
}

// ShouldRecurse always returns true in order to walk nested structs.
func (*TextUnmarshalerMangler) ShouldRecurse(reflect.StructField) bool {
	return true
}
