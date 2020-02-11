package transform

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/vimeo/dials/parsestring"
)

var (
	zeroStr    = ""
	strPtrType = reflect.TypeOf(&zeroStr)
)

// StringCastingMangler mangles config struct fields into string types, then
// unmangles the filled-in fields back to the original types, in order to
// abstract away the details of type conversion from sources.
type StringCastingMangler struct{}

// Mangle changes the type of the provided StructField to string.
func (*StringCastingMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	sf.Type = strPtrType
	return []reflect.StructField{sf}, nil
}

// Unmangle casts the string value in the mangled config struct to the type in
// the original struct.
func (*StringCastingMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {
	// Get the string value that was set on the mangled StructField in order to
	// cast it to the type in the original StructField, or return with a zero
	// value of the original StructField's type if no string value was set.
	strPtrInterface := vs[0].Value.Interface()
	var nilStrPtr *string
	if strPtrInterface == nilStrPtr {
		return reflect.Zero(sf.Type), nil
	}
	str := *(strPtrInterface.(*string))

	// Handle StructField types that implement the TextUnmarshaler interface
	// (defining their own way for an object to unmarshal a textual
	// representation of itself).
	textUnmarshalerType := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	if reflect.PtrTo(sf.Type).Implements(textUnmarshalerType) { // If type is concrete type implementing TextUnmarshaler, e.g. net.IP
		textUnmarshalerPtr := reflect.New(sf.Type)
		val := textUnmarshalerPtr.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(str))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v", sf.Type)
		}
		return textUnmarshalerPtr.Elem(), nil
	} else if sf.Type.Implements(textUnmarshalerType) { // If type is pointer to type implementing TextUnmarshaler, e.g. *net.IP
		textUnmarshalerPtr := reflect.New(sf.Type.Elem())
		val := textUnmarshalerPtr.Interface().(encoding.TextUnmarshaler)
		err := val.UnmarshalText([]byte(str))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Error unmarshaling text into type %+v", sf.Type)
		}
		return textUnmarshalerPtr, nil
	}

	// If the StructField type wasn't a TextUnmarshaler, set what type we'll be
	// casting to. All types in these StructFields from user-defined config
	// struct types, except for slices and maps, have been pointerified so that
	// we can distinguish nil values from zero values. For all types except
	// slices and maps, we get the pointed-to concrete type in order to deal
	// with it rather than pointer types themselves, for readability.
	var castTo reflect.Type
	switch sf.Type.Kind() {
	case reflect.Slice, reflect.Map:
		castTo = sf.Type
	default:
		castTo = sf.Type.Elem()
	}

	return parsestring.ParseString(str, castTo)
}

// ShouldRecurse always returns true in order to walk nested structs.
func (*StringCastingMangler) ShouldRecurse(reflect.StructField) bool {
	return true
}
