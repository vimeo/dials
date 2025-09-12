package dials

import (
	"fmt"
	"go/token"
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	type sInt struct{ J int }
	type sBool struct{ J bool }
	type RefCycleJ struct {
		B  bool
		Bs *bool
		J  *RefCycleJ
	}
	type RefCycleMapJ struct {
		B  bool
		Bs *bool
		J  map[string]*RefCycleMapJ
		Js map[string]*RefCycleMapJ
	}

	for name, inst := range map[string]struct {
		i     any
		check func(t testing.TB, in, out any)
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
			C any
		}{J: true, C: &sInt{J: 3}}},
		"bool_with_interface_int": {i: struct {
			J bool
			C any
		}{J: true, C: 1}},
		"bool_with_interface_map": {i: struct {
			J bool
			M any
		}{J: true,
			M: map[string]int{"foobarbaz": 123,
				"baz": 2,
				"answer to life, the universe and everything": 42},
		}},
		"bool_with_interface_slice": {i: struct {
			J bool
			S any
		}{J: true,
			S: []string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_interface_array": {i: struct {
			J bool
			S any
		}{J: true,
			S: [...]string{"foobarbaz",
				"baz",
				"answer to life, the universe and everything"},
		}},
		"bool_with_interface_in_interface": {i: struct {
			J bool
			S any
		}{J: true,
			S: struct{ L any }{L: &sInt{J: 42}},
		}},
		"bool_with_ptr_interface_in_interface": {i: struct {
			J bool
			S any
		}{J: true,
			S: &struct{ L any }{L: &sInt{J: 42}},
		}},
		"bool_with_interface_nested_slice": {i: struct {
			J bool
			S any
		}{J: true,
			S: []sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"nested_with_interface_ptr_bool": {i: struct{ K any }{K: &sBool{J: true}}},
		"bool_with_interface_nested_array": {i: struct {
			J bool
			S any
		}{J: true,
			S: [...]sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_interface_nested_ptr_array": {i: struct {
			J bool
			S any
		}{J: true,
			S: [...]*sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_interface_nested_ptr_slice": {i: struct {
			J bool
			S any
		}{J: true,
			S: []*sInt{{1}, {2}, {3}, {4}, {5}},
		}},
		"bool_with_internal_ref": {i: func() *struct {
			B  bool
			Bs *bool
		} {
			// note: this must be a pointer-type so the
			// struct-value itself is addressable, since otherwise
			// the fields aren't addressable either.
			b := &struct {
				B  bool
				Bs *bool
			}{
				B: true,
			}
			b.Bs = &b.B
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				b := out.(*struct {
					B  bool
					Bs *bool
				})
				if &b.B != b.Bs {
					t.Errorf("referential consistency violation: b.Bs: got %p; want: %p; at %p",
						b.Bs, &b.B, &b.Bs)
				}
			},
		},
		"bool_ptrs_with_common_ref": {i: func() *struct {
			B  *bool
			Bs *bool
		} {
			z := true
			// note: this must be a pointer-type so the
			// struct-value itself is addressable, since otherwise
			// the fields aren't addressable either.
			b := &struct {
				B  *bool
				Bs *bool
			}{
				B:  &z,
				Bs: &z,
			}
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				b := out.(*struct {
					B  *bool
					Bs *bool
				})
				if b.B != b.Bs {
					t.Errorf("referential consistency violation: b.Bs and b.B should match: got %p; want: %p",
						b.Bs, b.B)
				}
			},
		},
		"bool_ptrs_with_common_ref_unaddressable_struct": {i: func() struct {
			B  *bool
			Bs *bool
		} {
			z := true
			// note: this must be a pointer-type so the
			// struct-value itself is addressable, since otherwise
			// the fields aren't addressable either.
			b := struct {
				B  *bool
				Bs *bool
			}{
				B:  &z,
				Bs: &z,
			}
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				b := out.(struct {
					B  *bool
					Bs *bool
				})
				if b.B != b.Bs {
					t.Errorf("referential consistency violation: b.Bs and b.B should match: got %p; want: %p",
						b.Bs, b.B)
				}
			},
		},
		"struct_with_ptr_ref_cycle": {i: func() any {
			b := &RefCycleJ{
				B: true,
			}
			b.Bs = &b.B
			b.J = b
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				b := out.(*RefCycleJ)
				if &b.B != b.Bs {
					t.Errorf("referential consistency violation: b.Bs: got %p; want: %p; at %p",
						b.Bs, &b.B, &b.Bs)
				}
				if b != b.J {
					t.Errorf("referential consistency violation: b.J: got %p; want: %p; at %p",
						b.J, b, &b.J)

				}
			},
		},
		"struct_with_ptr_ref_to_unexported_field": {i: func() any {
			b := &struct {
				r RefCycleJ
				R *RefCycleJ
			}{
				r: RefCycleJ{
					B:  false,
					Bs: nil,
					J:  &RefCycleJ{},
				},
			}
			b.r.Bs = &b.r.B
			b.r.J = &b.r
			b.R = &b.r
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				i := in.(*struct {
					r RefCycleJ
					R *RefCycleJ
				})
				b := out.(*struct {
					r RefCycleJ
					R *RefCycleJ
				})
				// We can't do anything about values within
				// unexported fields
				if &i.r.B != b.r.Bs {
					t.Errorf("referential consistency violation: b.r.Bs: got %p; want: %p; at %p",
						b.r.Bs, &i.r.B, &b.r.Bs)
				}
				if &i.r != b.r.J {
					t.Errorf("referential consistency violation: b.r.J: got %p; want: %p",
						b.r.J, &i.r)

				}
			},
		},
		"struct_with_map_ref_cycle": {i: func() any {
			b := &RefCycleMapJ{
				B: true,
			}
			b.Bs = &b.B
			b.J = map[string]*RefCycleMapJ{"fimbat": b}
			b.Js = b.J
			return b
		}(),
			check: func(t testing.TB, in, out any) {
				b := out.(*RefCycleMapJ)
				if &b.B != b.Bs {
					t.Errorf("referential consistency violation: b.Bs: got %p; want: %p; at %p",
						b.Bs, &b.B, &b.Bs)
				}
				if b != b.J["fimbat"] {
					t.Errorf("referential consistency violation: b.J: got %p; want: %p",
						b.J, b)

				}
				// verify that the maps have the same underlying pointer
				if reflect.ValueOf(b.J).Pointer() != reflect.ValueOf(b.Js).Pointer() {
					t.Errorf("map referential consistency violation: b.J != b.Js: got %d; want: %d",
						reflect.ValueOf(b.J).Pointer(), reflect.ValueOf(b.Js).Pointer())
				}
			},
		},
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
			verifyDifferentPointers(t, nil, "", reflect.ValueOf(entry.i), out)
			if entry.check != nil {
				entry.check(t, entry.i, iface)
			}
		})

	}
}

func verifyDifferentPointers(t testing.TB, seenPtrs map[uintptr]struct{}, fname string, in, out reflect.Value) {
	t.Helper()
	if !in.IsValid() {
		t.Fatalf("invalid input value at %s", fname)
	}
	if !out.IsValid() {
		t.Fatalf("invalid output value at %s", fname)
	}

	if seenPtrs == nil {
		seenPtrs = make(map[uintptr]struct{}, 1)
	}

	switch in.Kind() {
	case reflect.Pointer:
		if in.Pointer() == out.Pointer() {
			t.Errorf("field %s pointer preserved", fname)
		}
		if in.IsNil() {
			return
		}
		if _, ok := seenPtrs[in.Pointer()]; ok {
			// we've already checked this input pointer
			return
		}
		seenPtrs[in.Pointer()] = struct{}{}
		verifyDifferentPointers(t, seenPtrs, "(*"+fname+")", in.Elem(), out.Elem())
	case reflect.Interface:
		if in.IsNil() {
			return
		}
		verifyDifferentPointers(t, seenPtrs, fname, in.Elem(), out.Elem())
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
			verifyDifferentPointers(t, seenPtrs, fmt.Sprintf("%s[%d]", fname, z), in.Index(z), out.Index(z))
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

			verifyDifferentPointers(t, seenPtrs, fmt.Sprintf("%s[%s]", fname, k), inElem, outElem)
		}
	case reflect.Struct:
		it := in.Type()
		for z := 0; z < in.NumField(); z++ {
			ifield := it.Field(z)
			if !token.IsExported(ifield.Name) {
				// field's not exported, so we can't recurse
				// into it to set values/pull-out-pointers
				// anyway.
				return
			}
			inFieldType := in.Type().Field(z)
			inField := in.Field(z)
			outField := out.Field(z)
			verifyDifferentPointers(t, seenPtrs, fname+"."+inFieldType.Name, inField, outField)
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
