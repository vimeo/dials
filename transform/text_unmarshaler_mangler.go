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
		if strPtr == nil {
			return v, nil
		}
		val := v.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(*strPtr))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v", sf.Type)
		}
		return v, nil
	})
	// return reflect.Value{}, nil
}

// OldUnmangle casts the string value in the mangled config struct to the type in
// the original struct.
func (*TextUnmarshalerMangler) OldUnmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {

	if reflect.PtrTo(sf.Type).Implements(textUnmarshalerType) { // If type is concrete type implementing TextUnmarshaler, e.g. net.IP
		strVal := *(vs[0].Value.Interface().(*string))
		textUnmarshalerPtr := reflect.New(sf.Type)
		val := textUnmarshalerPtr.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(strVal))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v", sf.Type)
		}
		return textUnmarshalerPtr.Elem(), nil
	} else if sf.Type.Implements(textUnmarshalerType) { // If type is pointer to type implementing TextUnmarshaler, e.g. *net.IP
		strVal := *(vs[0].Value.Interface().(*string))
		textUnmarshalerPtr := reflect.New(sf.Type.Elem())
		val := textUnmarshalerPtr.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(strVal))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v", sf.Type)
		}
		return textUnmarshalerPtr, nil
	}

	// it's not a TextUnmarshaler, so just return an get out...
	return vs[0].Value, nil
}

// ShouldRecurse always returns true in order to walk nested structs.
func (*TextUnmarshalerMangler) ShouldRecurse(reflect.StructField) bool {
	return true
}
