package dials

import (
	"fmt"
	"go/token"
	"reflect"
)

// takes a concrete value for in and returns a concrete deep copy Value
func realDeepCopy(in any) reflect.Value {
	v := reflect.ValueOf(in)
	return deepCopyValue(v)
}

func deepCopyValue(v reflect.Value) reflect.Value {
	d := newDeepCopier()
	return d.deepCopyValue(v)
}

type ptrKey struct {
	ptr uintptr
	typ reflect.Type
}

func newDeepCopier() *deepCopier {
	return &deepCopier{
		ptrMap: map[ptrKey]reflect.Value{},
		mapMap: map[uintptr]reflect.Value{},
	}
}

type deepCopier struct {
	// map from input pointer to output pointer to handle reference-cycles
	// and splitting pointers to the same object.
	ptrMap map[ptrKey]reflect.Value

	// map from input map-pointer to output-map to handle
	// reference cycles.
	mapMap map[uintptr]reflect.Value
}

func (d *deepCopier) deepCopyValue(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	d.deepCopy(v, out)
	return out
}

func (d *deepCopier) registerPair(in, out reflect.Value) {
	if in.CanAddr() && out.CanAddr() {
		p := in.Addr()
		pKey := ptrKey{ptr: p.Pointer(), typ: p.Type()}
		d.ptrMap[pKey] = out.Addr()
	}
}

// takes a concrete value for in, an assignable value in out.
func (d *deepCopier) deepCopy(in, out reflect.Value) {

	// Start with setting the value directly if possible, so we get private
	// fields.
	// Note that this should copy channels and functions over such that
	// they have the same identity
	if out.CanSet() {
		out.Set(in)
	}

	d.registerPair(in, out)

	switch in.Kind() {
	case reflect.Struct:
		d.deepCopyStruct(in, out)
	case reflect.Pointer:
		d.deepCopyPtr(in, out)
	case reflect.Interface:
		d.deepCopyIface(in, out)
	case reflect.Map:
		d.deepCopyMap(in, out)
	case reflect.Slice:
		d.deepCopySlice(in, out)
	case reflect.Array:
		d.deepCopyArray(in, out)
	default:
	}
}

func (d *deepCopier) deepCopyIface(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	inElem := in.Elem()
	switch inElem.Kind() {
	case reflect.Pointer:
		newVal := reflect.New(inElem.Type().Elem())
		out.Set(newVal)
		d.deepCopy(inElem.Elem(), newVal.Elem())
		return
	case reflect.Struct:
		newVal := reflect.New(inElem.Type())
		d.deepCopy(inElem, newVal.Elem())
		out.Set(newVal.Elem())
		return
	case reflect.Map:
		if inElem.IsNil() {
			return
		}
		out.Set(reflect.MakeMapWithSize(inElem.Type(), inElem.Len()))
		d.deepCopy(inElem, out.Elem())
		return
	case reflect.Slice:
		if inElem.IsNil() {
			return
		}
		out.Set(reflect.MakeSlice(inElem.Type(), inElem.Len(), inElem.Cap()))
		d.deepCopy(inElem, out.Elem())
		return
	case reflect.Array:
		newVal := reflect.New(inElem.Type())
		d.deepCopyArray(inElem, newVal.Elem())
		out.Set(newVal.Elem())
		return
	}
	// not a pointer, struct or map
	out.Set(inElem)
}
func (d *deepCopier) deepCopyPtr(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	pKey := ptrKey{ptr: in.Pointer(), typ: in.Type()}
	if ov, ok := d.ptrMap[pKey]; ok {
		out.Set(ov)
		// The deep part of the copying has already been taken care of
		return
	}

	inType := in.Type()
	newVal := reflect.New(inType.Elem())
	out.Set(newVal)
	d.ptrMap[pKey] = out
	d.deepCopy(in.Elem(), out.Elem())
}

// deepCopyStruct does a deep-copy of the passed struct-type from in into out.
func (d *deepCopier) deepCopyStruct(in, out reflect.Value) {
	for i := 0; i < in.NumField(); i++ {
		ifld := in.Type().Field(i)
		f := in.Field(i)
		of := out.Field(i)
		if !token.IsExported(ifld.Name) {
			// The field is not exported, and Go's reflect package
			// goes to great lengths to consider values from such
			// fields poisoned. Don't try to bypass it, since we're
			// dealing with config structs which are assembled from
			// different sources, not general-purpose values.
			continue
		}
		d.deepCopy(f, of)
	}
}

// deepCopySlice allocates a new slice copying values from in, and assigns it
// to out.
func (d *deepCopier) deepCopySlice(in, out reflect.Value) {
	if in.Kind() != reflect.Slice || out.Kind() != reflect.Slice {
		panic(fmt.Errorf("unexpected type: in: %s; out: %s", in.Type(), out.Type()))
	}
	if in.IsNil() {
		return
	}

	if (out.IsNil() || out.Pointer() == in.Pointer()) && out.CanSet() {
		out.Set(reflect.MakeSlice(in.Type(), in.Len(), in.Cap()))
	}
	// Copy the entire backing array
	d.deepCopyArray(in.Slice(0, in.Cap()), out.Slice(0, out.Cap()))
}

// deepCopyArray copies values in an array, or a pre-allocated slice.
func (d *deepCopier) deepCopyArray(in, out reflect.Value) {
	if in.Type() != out.Type() {
		panic(fmt.Errorf("mismatched array-ish types: in: %s; out: %s", in.Type(), out.Type()))
	}
	switch in.Kind() {
	case reflect.Array, reflect.Slice:
	default:
		panic(fmt.Errorf("non-array-ish types: in: %s; out: %s", in.Type(), out.Type()))
	}
	for z := 0; z < in.Len(); z++ {
		d.deepCopy(in.Index(z), out.Index(z))
	}
}

func (d *deepCopier) deepCopyMap(in, out reflect.Value) {
	if in.IsNil() {
		return
	}
	if mv, ok := d.mapMap[in.Pointer()]; ok && out.CanSet() {
		// We've seen this map before, let's take advantage of it.
		out.Set(mv)
		return
	}
	// Mark this map's backing pointer as handled, so back-references get
	// handled properly if they occur in values.
	d.mapMap[in.Pointer()] = out

	if (out.IsNil() || out.Pointer() == in.Pointer()) && out.CanSet() {
		out.Set(reflect.MakeMapWithSize(in.Type(), in.Len()))
	}
	iter := in.MapRange()
	for iter.Next() {
		oldKey := iter.Key()
		oldVal := iter.Value()
		newKey := reflect.New(oldKey.Type()).Elem()
		newVal := reflect.New(oldVal.Type()).Elem()
		d.deepCopy(oldKey, newKey)
		d.deepCopy(oldVal, newVal)
		out.SetMapIndex(newKey, newVal)
	}
}
