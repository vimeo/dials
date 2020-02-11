package dials

import (
	"fmt"
	"reflect"
)

// takes a concrete value for in and returns a concrete deep copy Value
func realDeepCopy(in interface{}) reflect.Value {
	v := reflect.ValueOf(in)
	out := reflect.New(v.Type()).Elem()
	deepCopy(v, out)
	return out
}

// takes a concrete value for in, an assignable value in out.
func deepCopy(in, out reflect.Value) {
	// Start with setting the value directly if possible, so we get private
	// fields.
	// Note that this should copy channels and functions over such that
	// they have the same identity
	if out.CanSet() {
		out.Set(in)
	}
	switch in.Kind() {
	case reflect.Struct:
		deepCopyStruct(in, out)
	case reflect.Ptr:
		deepCopyPtr(in, out)
	case reflect.Interface:
		deepCopyIface(in, out)
	case reflect.Map:
		deepCopyMap(in, out)
	case reflect.Slice:
		deepCopySlice(in, out)
	case reflect.Array:
		deepCopyArray(in, out)
	default:
	}
}

func deepCopyIface(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	inElem := in.Elem()
	switch inElem.Kind() {
	case reflect.Ptr:
		newVal := reflect.New(inElem.Type().Elem())
		out.Set(newVal)
		deepCopy(inElem.Elem(), newVal.Elem())
		return
	case reflect.Struct:
		newVal := reflect.New(inElem.Type())
		deepCopy(inElem, newVal.Elem())
		out.Set(newVal.Elem())
		return
	case reflect.Map:
		if inElem.IsNil() {
			return
		}
		out.Set(reflect.MakeMapWithSize(inElem.Type(), inElem.Len()))
		deepCopy(inElem, out.Elem())
		return
	case reflect.Slice:
		if inElem.IsNil() {
			return
		}
		out.Set(reflect.MakeSlice(inElem.Type(), inElem.Len(), inElem.Cap()))
		deepCopy(inElem, out.Elem())
		return
	case reflect.Array:
		newVal := reflect.New(inElem.Type())
		deepCopyArray(inElem, newVal.Elem())
		out.Set(newVal.Elem())
		return
	}
	// not a pointer, struct or map
	out.Set(inElem)
}
func deepCopyPtr(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	inType := in.Type()
	newVal := reflect.New(inType.Elem())
	out.Set(newVal)
	deepCopy(in.Elem(), out.Elem())
}

// deepCopyStruct does a deep-copy of the passed struct-type from in into out.
func deepCopyStruct(in, out reflect.Value) {
	for i := 0; i < in.NumField(); i++ {
		f := in.Field(i)
		of := out.Field(i)
		deepCopy(f, of)
	}
}

// deepCopySlice allocates a new slice copying values from in, and assigns it
// to out.
func deepCopySlice(in, out reflect.Value) {
	if in.Kind() != reflect.Slice || out.Kind() != reflect.Slice {
		panic(fmt.Errorf("unexpected type: in: %s; out: %s", in.Type(), out.Type()))
	}
	if in.IsNil() {
		return
	}
	if (out.IsNil() || out.Pointer() == in.Pointer()) && out.CanSet() {
		out.Set(reflect.MakeSlice(in.Type(), in.Len(), in.Cap()))
	}
	deepCopyArray(in, out)
}

// deepCopyArray copies values in an array, or a pre-allocated slice.
func deepCopyArray(in, out reflect.Value) {
	if in.Type() != out.Type() {
		panic(fmt.Errorf("mismatched array-ish types: in: %s; out: %s", in.Type(), out.Type()))
	}
	switch in.Kind() {
	case reflect.Array, reflect.Slice:
	default:
		panic(fmt.Errorf("non-array-ish types: in: %s; out: %s", in.Type(), out.Type()))
	}
	for z := 0; z < in.Len(); z++ {
		deepCopy(in.Index(z), out.Index(z))
	}
}

func deepCopyMap(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	if (out.IsNil() || out.Pointer() == in.Pointer()) && out.CanSet() {
		out.Set(reflect.MakeMapWithSize(in.Type(), in.Len()))
	}
	iter := in.MapRange()
	for iter.Next() {
		oldKey := iter.Key()
		oldVal := iter.Value()
		newKey := reflect.New(oldKey.Type()).Elem()
		newVal := reflect.New(oldVal.Type()).Elem()
		deepCopy(oldKey, newKey)
		deepCopy(oldVal, newVal)
		out.SetMapIndex(newKey, newVal)
	}
}
