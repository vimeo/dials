package dials

import (
	"reflect"
	"testing"
	"time"
)

func TestOverlay(t *testing.T) {
	type sInt struct{ J int }
	type sBool struct{ J bool }
	three := int(3)
	True := true
	sampleChan := make(chan struct{})
	now := time.Now()

	for name, inst := range map[string]struct {
		// Note: base must be a pointer-type to make it mutable
		base     any
		overlay  any
		expected any
	}{
		"just_int_overlayed": {base: &sInt{J: 2},
			overlay:  struct{ J *int }{J: &three},
			expected: sInt{J: 3}},
		"just_bool_overlayed": {base: &sBool{J: false},
			overlay:  struct{ J *bool }{J: &True},
			expected: sBool{J: true}},
		"bool_overlayed_nil_overlay": {base: &sBool{J: false},
			overlay:  struct{ J *bool }{J: nil},
			expected: sBool{J: false}},
		"bool_overlayed_interface_bool_overlay": {base: &struct{ J any }{J: false},
			overlay:  struct{ J *bool }{J: nil},
			expected: struct{ J any }{J: false}},
		"bool_with_chan": {base: &struct {
			J bool
			C chan struct{}
		}{J: true, C: sampleChan},
			overlay: struct {
				J *bool
				C chan struct{}
			}{J: &True, C: sampleChan},
			expected: struct {
				J bool
				C chan struct{}
			}{J: true, C: sampleChan},
		},
		"bool_private_int": {base: &struct {
			J bool
			c int
		}{J: true, c: 3},
			overlay: struct {
				J *bool
			}{J: &True},
			expected: struct {
				J bool
				c int
			}{J: true, c: 3},
		},
		"nested_with_ptr_bool": {base: &struct{ K *sBool }{K: &sBool{J: true}},
			overlay:  struct{ K *struct{ J *bool } }{K: &struct{ J *bool }{J: &True}},
			expected: struct{ K *sBool }{K: &sBool{J: true}},
		},
		"nested_with_ptr_bool_base_nil": {base: &struct{ K *sBool }{K: nil},
			overlay:  struct{ K *struct{ J *bool } }{K: &struct{ J *bool }{J: &True}},
			expected: struct{ K *sBool }{K: &sBool{J: true}},
		},
		"one_deep_with_nil_overlay_time": {
			base: &struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{},
			expected: struct {
				I sInt
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
		},
		"one_deep_with_overlay_time": {
			base: &struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T time.Time
			}{I: sInt{
				J: 42,
			}},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T time.Time
			}{I: sInt{
				J: 3,
			}, T: now},
		},
		"one_deep_with_overlay_pointer_time": {
			base: &struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T *time.Time
			}{I: sInt{
				J: 42,
			}},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T *time.Time
			}{I: sInt{
				J: 3,
			}, T: &now},
		},
		"overlay_pointer_time_no_nil_ptr": {
			base: &struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T *time.Time
			}{I: sInt{
				J: 42,
			}, T: &time.Time{}},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T *time.Time
			}{I: sInt{
				J: 3,
			}, T: &now},
		},
		"empty_overlayed_interface_overlay": {base: &struct{ J any }{J: false},
			overlay:  struct{ J any }{J: nil},
			expected: struct{ J any }{J: false}},
		"nested_with_iface_ptr_bool_base_int": {
			base:     &struct{ K any }{K: int(39)},
			overlay:  struct{ K *struct{ J *bool } }{K: &struct{ J *bool }{J: &True}},
			expected: struct{ K any }{K: &struct{ J *bool }{J: &True}},
		},
		"nested_with_iface_ptr_bool_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K *struct{ J *bool } }{K: &struct{ J *bool }{J: &True}},
			expected: struct{ K any }{K: &struct{ J *bool }{J: &True}},
		},
		"nested_with_ifaces_ptr_bool_base_int": {
			base:     &struct{ K any }{K: int(39)},
			overlay:  struct{ K any }{K: &struct{ J *bool }{J: &True}},
			expected: struct{ K any }{K: &struct{ J *bool }{J: &True}},
		},
		"nested_with_iface_bool_base_int": {
			base:     &struct{ K any }{K: int(39)},
			overlay:  struct{ K struct{ J bool } }{K: struct{ J bool }{J: true}},
			expected: struct{ K any }{K: struct{ J bool }{J: true}},
		},
		"nested_with_iface_bool_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K struct{ J bool } }{K: struct{ J bool }{J: true}},
			expected: struct{ K any }{K: struct{ J bool }{J: true}},
		},
		"nested_with_ifaces_bool_base_int": {
			base:     &struct{ K any }{K: int(39)},
			overlay:  struct{ K any }{K: sBool{J: true}},
			expected: struct{ K any }{K: sBool{J: true}},
		},
		"one_deep_with_iface_resolved_overlay_time": {
			base: &struct {
				I any
				// time.Time implements TextUnmarshaler
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{},
			expected: struct {
				I any
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
		},
		"one_deep_with_iface_overlay_time": {
			base: &struct {
				I any
				// time.Time implements TextUnmarshaler
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
			overlay: struct {
				I any
				T *time.Time
			}{},
			expected: struct {
				I any
				T time.Time
			}{I: sInt{
				J: 42,
			}, T: now},
		},
		"one_deep_with_iface_overlay_time_alliface_nonnil": {
			base: &struct {
				I sInt
				T any
			}{I: sInt{
				J: 42,
			}},
			overlay: struct {
				I *struct{ J *int }
				T any
			}{I: &struct{ J *int }{
				J: &three,
			}, T: now},
			expected: struct {
				I sInt
				T any
			}{I: sInt{
				J: 3,
			}, T: now},
		},
		"one_deep_with_iface_overlay_time_nonnil": {
			base: &struct {
				I sInt
				T any
			}{I: sInt{
				J: 42,
			}},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T any
			}{I: sInt{
				J: 3,
			}, T: &now},
		},
		"one_deep_with_overlay_iface_pointer_time": {
			base: &struct {
				I sInt
				T any
			}{I: sInt{
				J: 42,
			}},
			overlay: struct {
				I *struct{ J *int }
				T *time.Time
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T any
			}{I: sInt{
				J: 3,
			}, T: &now},
		},
		"overlay_pointer_time_iface_no_nil_ptr": {
			base: &struct {
				I sInt
				T any
			}{I: sInt{
				J: 42,
			}, T: &time.Time{}},
			overlay: struct {
				I *struct{ J *int }
				T any
			}{I: &struct{ J *int }{
				J: &three,
			}, T: &now},
			expected: struct {
				I sInt
				T any
			}{I: sInt{
				J: 3,
			}, T: &now},
		},
		"map_simple_base_nil": {
			base:     &struct{ K map[string]string }{K: nil},
			overlay:  struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
			expected: struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
		},
		// We're not trying to merge maps or slices
		"map_simple_base_not_nil": {
			base:     &struct{ K map[string]string }{K: map[string]string{"foo": "baz"}},
			overlay:  struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
			expected: struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
		},
		"map_iface_base_not_nil": {
			base:     &struct{ K any }{K: map[string]string{"foo": "baz"}},
			overlay:  struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
			expected: struct{ K any }{K: map[string]string{"foo": "bar"}},
		},
		"map_iface_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K map[string]string }{K: map[string]string{"foo": "bar"}},
			expected: struct{ K any }{K: map[string]string{"foo": "bar"}},
		},
		"slice_simple_base_nil": {
			base:     &struct{ K []string }{K: nil},
			overlay:  struct{ K []string }{K: []string{"bar"}},
			expected: struct{ K []string }{K: []string{"bar"}},
		},
		// We're not trying to merge maps or slices
		"slice_simple_base_not_nil": {
			base:     &struct{ K []string }{K: []string{"baz"}},
			overlay:  struct{ K []string }{K: []string{"bar"}},
			expected: struct{ K []string }{K: []string{"bar"}},
		},
		"slice_iface_base_not_nil": {
			base:     &struct{ K any }{K: []string{"baz"}},
			overlay:  struct{ K []string }{K: []string{"bar"}},
			expected: struct{ K any }{K: []string{"bar"}},
		},
		"slice_iface_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K []string }{K: []string{"bar"}},
			expected: struct{ K any }{K: []string{"bar"}},
		},
		// We're not trying to merge arrays
		"array_simple": {
			base:     &struct{ K [1]string }{K: [...]string{"baz"}},
			overlay:  struct{ K [1]string }{K: [...]string{"bar"}},
			expected: struct{ K [1]string }{K: [...]string{"bar"}},
		},
		"array_iface_base_not_nil": {
			base:     &struct{ K any }{K: [...]string{"baz"}},
			overlay:  struct{ K [1]string }{K: [...]string{"bar"}},
			expected: struct{ K any }{K: [...]string{"bar"}},
		},
		"array_iface_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K [1]string }{K: [...]string{"bar"}},
			expected: struct{ K any }{K: [...]string{"bar"}},
		},
		"array_iface_ptr_base_concrete_not_nil": {
			base:     &struct{ K any }{K: [...]string{"baz"}},
			overlay:  struct{ K *[1]string }{K: &[...]string{"bar"}},
			expected: struct{ K any }{K: [...]string{"bar"}},
		},
		"array_iface_ptr_base_not_nil": {
			base:     &struct{ K any }{K: &[...]string{"baz"}},
			overlay:  struct{ K *[1]string }{K: &[...]string{"bar"}},
			expected: struct{ K any }{K: &[...]string{"bar"}},
		},
		"array_iface_ptr_base_nil": {
			base:     &struct{ K any }{K: nil},
			overlay:  struct{ K *[1]string }{K: &[...]string{"bar"}},
			expected: struct{ K any }{K: &[...]string{"bar"}},
		},
		"array_iface_ptr_base_nil_with_type": {
			base:     &struct{ K any }{K: (*[1]string)(nil)},
			overlay:  struct{ K *[1]string }{K: &[...]string{"bar"}},
			expected: struct{ K any }{K: &[...]string{"bar"}},
		},
	} {
		entry := inst
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			base := reflect.ValueOf(entry.base).Elem()
			overlay := reflect.ValueOf(entry.overlay)
			o := newOverlayer()
			if err := o.overlayStruct(base, overlay); err != nil {
				t.Fatalf("failed to overlay struct: %s",
					err)
			}

			if !reflect.DeepEqual(base.Interface(), entry.expected) {
				t.Errorf("got: %#v, expected: %#v",
					base, entry.expected)
			}
		})

	}
}
