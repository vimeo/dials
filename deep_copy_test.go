package dials

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	type sInt struct{ J int }
	type sBool struct{ J bool }

	for name, inst := range map[string]struct {
		i interface{}
	}{
		"just_int": {i: sInt{J: 3}},
		"one_public_one_private_int": {i: struct {
			J int
			p int
		}{J: 3, p: 42}},
		"just_int_ptr":     {i: &sInt{J: 3}},
		"just_str":         {i: struct{ J string }{J: "foobarbaz"}},
		"just_bool":        {i: sBool{J: true}},
		"nested_with_bool": {i: struct{ K sBool }{K: sBool{J: true}}},
		"bool_with_chan": {i: struct {
			J bool
			C chan struct{}
		}{J: true, C: make(chan struct{})}},
		"bool_with_map": {i: struct {
			J bool
			M map[string]int
		}{J: true,
			M: map[string]int{"foobarbaz": 123,
				"baz": 2,
				"answer to life, the universe and everything": 42},
		}},
		"bool_with_slice": {i: struct {
			J bool
			S []string
		}{J: true,
			S: []string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_array": {i: struct {
			J bool
			S [3]string
		}{J: true,
			S: [...]string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_nested_slice": {i: struct {
			J bool
			S []sInt
		}{J: true,
			S: []sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_nested_array": {i: struct {
			J bool
			S [5]sInt
		}{J: true,
			S: [...]sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"nested_with_ptr_bool": {i: struct{ K *sBool }{K: &sBool{J: true}}},
		"bool_with_private_int": {i: struct {
			J bool
			c int
		}{J: true, c: 0}},
		"bool_with_interface_struct_ptr": {i: struct {
			J bool
			C interface{}
		}{J: true, C: &sInt{J: 3}}},
		"bool_with_interface_int": {i: struct {
			J bool
			C interface{}
		}{J: true, C: 1}},
		"bool_with_interface_map": {i: struct {
			J bool
			M interface{}
		}{J: true,
			M: map[string]int{"foobarbaz": 123,
				"baz": 2,
				"answer to life, the universe and everything": 42},
		}},
		"bool_with_interface_slice": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: []string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_interface_array": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: [...]string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_interface_in_interface": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: struct{ L interface{} }{L: &sInt{J: 42}},
		}},
		"bool_with_ptr_interface_in_interface": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: &struct{ L interface{} }{L: &sInt{J: 42}},
		}},
		"bool_with_interface_nested_slice": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: []sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"nested_with_interface_ptr_bool": {i: struct{ K interface{} }{K: &sBool{J: true}}},
		"bool_with_interface_nested_array": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: [...]sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_interface_nested_ptr_array": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: [...]*sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_interface_nested_ptr_slice": {i: struct {
			J bool
			S interface{}
		}{J: true,
			S: []*sInt{{1}, {2}, {3}, {4}, {5}},
		}},
	} {
		entry := inst
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out := realDeepCopy(entry.i)
			iface := out.Interface()
			if !reflect.DeepEqual(entry.i, iface) {
				t.Errorf("unequal values in: %+v, out: %+v",
					entry.i, iface)
			}
			verifyDifferentPointers(t, "", reflect.ValueOf(entry.i), out)
		})

	}
}

func verifyDifferentPointers(t testing.TB, fname string, in, out reflect.Value) {
	t.Helper()
	if !in.IsValid() {
		t.Fatalf("invalid input value at %s", fname)
	}
	if !out.IsValid() {
		t.Fatalf("invalid output value at %s", fname)
	}
	switch in.Kind() {
	case reflect.Ptr:
		if in.Pointer() == out.Pointer() {
			t.Errorf("field %s pointer preserved", fname)
		}
		if in.IsNil() {
			return
		}
		verifyDifferentPointers(t, "(*"+fname+")", in.Elem(), out.Elem())
	case reflect.Interface:
		if in.IsNil() {
			return
		}
		verifyDifferentPointers(t, fname, in.Elem(), out.Elem())
	case reflect.Slice:
		if in.IsNil() {
			return
		}
		if in.Pointer() == out.Pointer() {
			t.Errorf("field (slice) %s pointer preserved", fname)
		}
		fallthrough
	case reflect.Array:
		for z := 0; z < in.Len(); z++ {
			verifyDifferentPointers(t, fmt.Sprintf("%s[%d]", fname, z), in.Index(z), out.Index(z))
		}
	case reflect.Map:
		if in.IsNil() {
			return
		}
		if in.Pointer() == out.Pointer() {
			t.Errorf("field (map) %s pointer preserved", fname)
		}
		keys := in.MapKeys()
		for _, k := range keys {
			inElem := in.MapIndex(k)
			outElem := out.MapIndex(k)

			verifyDifferentPointers(t, fmt.Sprintf("%s[%s]", fname, k), inElem, outElem)
		}
	case reflect.Struct:
		for z := 0; z < in.NumField(); z++ {
			inFieldType := in.Type().Field(z)
			inField := in.Field(z)
			outField := out.Field(z)
			verifyDifferentPointers(t, fname+"."+inFieldType.Name, inField, outField)
		}
	case reflect.Chan, reflect.Func:
		if in.IsNil() {
			return
		}
		if in.Pointer() != out.Pointer() {
			t.Errorf("new %s pointer for field %s", in.Kind(), fname)
		}
	}
}
