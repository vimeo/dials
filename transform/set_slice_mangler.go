package transform

import (
	"fmt"
	"reflect"
)

var (
	emptyStructValue = reflect.ValueOf(struct{}{})
	emptyStructType  = emptyStructValue.Type()
)

// SetSliceMangler conveniently maps from sets (e.g. map[string]struct{}) to
// slices (e.g. []string)
type SetSliceMangler struct {
}

// Mangle changes the type of the provided StructField from a map[T]struct{} to
// []T.
func (*SetSliceMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	if sf.Type.Kind() == reflect.Map && sf.Type.Elem() == emptyStructType {
		sf.Type = reflect.SliceOf(sf.Type.Key())
	}

	return []reflect.StructField{sf}, nil
}

// Unmangle turns []T back into a map[T]struct{}.  Naturally, deduplication will
// occur.
func (*SetSliceMangler) Unmangle(sf reflect.StructField, vs []FieldValueTuple) (reflect.Value, error) {
	if sf.Type.Kind() != reflect.Map || sf.Type.Elem() != emptyStructType {
		return vs[0].Value, nil
	}

	slice := vs[0].Value

	if slice.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf(
			"expected slice to unmangle, instead got %s",
			slice.Kind(),
		)
	}

	if slice.Type().Elem() != sf.Type.Key() {
		return reflect.Value{}, fmt.Errorf(
			"expected slice elem to be %q, got %q",
			slice.Type().Elem(),
			sf.Type.Key(),
		)
	}

	// If the slice is nil, return a nil map.
	if slice.IsZero() {
		return reflect.Zero(sf.Type), nil
	}

	// slice.Len() could be larger if there are duplicates, but it's likely
	// a good place to start.
	set := reflect.MakeMapWithSize(sf.Type, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		set.SetMapIndex(slice.Index(i), emptyStructValue)
	}

	return set, nil
}

// ShouldRecurse always returns true in order to walk nested structs.
func (*SetSliceMangler) ShouldRecurse(reflect.StructField) bool {
	return true
}
