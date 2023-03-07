package ptrify

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPointerify(t *testing.T) {
	type sInt struct{ J int }
	type sIntPtr struct{ J *int }

	for name, inst := range map[string]struct {
		i        interface{}
		expected interface{}
	}{
		"shallow": {
			i:        sInt{},
			expected: sIntPtr{},
		},
		"one_deep": {
			i:        struct{ I sInt }{},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_with_private": {
			i: struct {
				I sInt
				l int
			}{},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_ptr": {
			i:        struct{ I sIntPtr }{},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_interface_empty": {
			i:        struct{ I interface{} }{},
			expected: struct{ I interface{} }{},
		},
		"one_deep_interface": {
			i:        struct{ I interface{} }{I: sInt{}},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_interface_non_nil_val": {
			i:        struct{ I interface{} }{I: &sInt{}},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_interface_nil_val": {
			i:        struct{ I interface{} }{I: (*sInt)(nil)},
			expected: struct{ I *struct{ J *int } }{},
		},
		"one_deep_interface_map": {
			i: struct{ I interface{} }{I: map[string]string{}},
			expected: struct {
				I map[string]string
			}{},
		},
		"one_deep_interface_nil_map": {
			i: struct{ I interface{} }{I: map[string]string(nil)},
			expected: struct {
				I map[string]string
			}{},
		},
		"one_deep_interface_slice": {
			i: struct{ I interface{} }{I: []string{}},
			expected: struct {
				I []string
			}{},
		},
		"one_deep_interface_nil_slice": {
			i: struct{ I interface{} }{I: []string(nil)},
			expected: struct {
				I []string
			}{},
		},
		"one_deep_interface_short_array": {
			i: struct{ I interface{} }{I: [...]string{"foobar", "baz"}},
			expected: struct {
				I *[2]string
			}{},
		},
		"one_deep_with_map": {
			i: struct {
				I sInt
				M map[string]int
			}{},
			expected: struct {
				I *struct{ J *int }
				M map[string]int
			}{},
		},
		"one_deep_with_time": {
			i: struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T time.Time
			}{},
			expected: struct {
				I *struct{ J *int }
				T *time.Time
			}{},
		},
		"one_deep_with_time_ptr": {
			i: struct {
				I sInt
				// time.Time implements TextUnmarshaler
				T *time.Time
			}{},
			expected: struct {
				I *struct{ J *int }
				T *time.Time
			}{},
		},
		"one_deep_with_slice": {
			i: struct {
				I sInt
				M []int
			}{},
			expected: struct {
				I *struct{ J *int }
				M []int
			}{},
		},
		"one_deep_with_array": {
			i: struct {
				I sInt
				M [3]int
			}{},
			expected: struct {
				I *struct{ J *int }
				M *[3]int
			}{},
		},
		"one_deep_with_chan_func": {
			i: struct {
				I sInt
				T func() bool
				C chan struct{}
			}{},
			expected: struct{ I *struct{ J *int } }{},
		},
		"three_deep_with_ptr": {
			i: struct {
				I struct{ J struct{ Q bool } }
				B struct{ L *struct{ S int32 } }
			}{},
			expected: struct {
				I *struct{ J *struct{ Q *bool } }
				B *struct{ L *struct{ S *int32 } }
			}{},
		},
		"three_deep_with_hypen": {
			i: struct {
				I struct {
					J struct{ Q bool } `dials:"-"`
					K int
				}
				B struct{ L *struct{ S int32 } }
			}{},
			expected: struct {
				I *struct{ K *int }
				B *struct{ L *struct{ S *int32 } }
			}{},
		},
	} {
		entry := inst
		t.Run(name, func(t *testing.T) {
			in := reflect.TypeOf(entry.i)
			v := reflect.ValueOf(entry.i)
			out := Pointerify(in, v)
			expected := reflect.TypeOf(entry.expected)
			if !out.ConvertibleTo(expected) {
				t.Errorf("unexpected pointerified type: got %s; expected %s",
					out.String(), expected.String())
			}
		})
	}
}

func TestEmbeddedPointerify(t *testing.T) {
	t.Parallel()
	type SString struct{ J *string }

	e := struct {
		SString
		E bool
	}{}

	in := reflect.TypeOf(e)
	v := reflect.ValueOf(e)
	out := Pointerify(in, v)

	assert.True(t, out.Field(0).Anonymous)
	assert.Equal(t, "SString", out.Field(0).Name)

	assert.False(t, out.Field(1).Anonymous)
	assert.Equal(t, "E", out.Field(1).Name)
}
