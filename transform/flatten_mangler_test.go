package transform

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestMangler(t *testing.T) {
	type foo struct {
		Location    string
		Coordinates int
	}

	type bar struct {
		Name   string
		Foobar *foo
	}

	b := bar{
		Name: "test",
		Foobar: &foo{
			Location:    "here",
			Coordinates: 64,
		},
	}

	testCases := []struct {
		name       string
		testStruct interface{}
		// modify will fill the flatten struct value after Mangling
		modify    func(reflect.Value)
		assertion func(interface{})
	}{
		{
			name:       "one member in struct of type int",
			testStruct: 32,
			modify: func(val reflect.Value) {
				i := 32
				val.Field(0).Set(reflect.ValueOf(&i))
			},
			assertion: func(i interface{}) {
				assert.Equal(t, 32, *i.(*int))
			},
		},
		{
			name:       "one member in struct of type map",
			testStruct: map[string]string{},
			modify: func(val reflect.Value) {
				m := map[string]string{
					"hello":   "world",
					"flatten": "unflatten",
				}
				val.Field(0).Set(reflect.ValueOf(m))
			},
			assertion: func(i interface{}) {
				m := map[string]string{
					"hello":   "world",
					"flatten": "unflatten",
				}
				assert.Equal(t, m, i.(map[string]string))
			},
		},
		{
			name: "one level nested struct unexposed fields",
			testStruct: struct {
				testInt    int
				testString string
				testBool   bool
			}{
				testInt:    42,
				testString: "hello world",
				testBool:   true,
			},
			modify: func(val reflect.Value) {},
			assertion: func(i interface{}) {
				// should be empty struct since none of the fields are exposed
				assert.Equal(t, struct{}{}, *i.(*struct{}))
			},
		},
		{
			name: "one level nested struct exposed fields",
			testStruct: struct {
				TestInt    int
				TestString string
				TestBool   bool
			}{
				TestInt:    42,
				TestString: "hello world",
				TestBool:   true,
			},
			modify: func(val reflect.Value) {
				i := 42
				s := "hello world"
				b := true

				val.Field(0).Set(reflect.ValueOf(&i))
				val.Field(1).Set(reflect.ValueOf(&s))
				val.Field(2).Set(reflect.ValueOf(&b))
			},
			assertion: func(i interface{}) {
				in := 42
				s := "hello world"
				b := true

				st := struct {
					TestInt    *int
					TestString *string
					TestBool   *bool
				}{
					TestInt:    &in,
					TestString: &s,
					TestBool:   &b,
				}
				assert.Equal(t, st, *i.(*struct {
					TestInt    *int
					TestString *string
					TestBool   *bool
				}))
			},
		},
		{
			name:       "multilevel nested struct",
			testStruct: b,
			modify: func(val reflect.Value) {
				s1 := "test"
				s2 := "here"
				i := 64

				val.Field(0).Set(reflect.ValueOf(&s1))
				val.Field(1).Set(reflect.ValueOf(&s2))
				val.Field(2).Set(reflect.ValueOf(&i))
			},
			assertion: func(i interface{}) {
				// all the fields are pointerified because of call to Pointerify
				s1 := "test"
				s2 := "here"
				i1 := 64
				b := struct {
					Name   *string
					Foobar *struct {
						Location    *string
						Coordinates *int
					}
				}{
					Name: &s1,
					Foobar: &struct {
						Location    *string
						Coordinates *int
					}{
						Location:    &s2,
						Coordinates: &i1,
					},
				}
				assert.Equal(t, &b, i)
			},
		},
	}

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			itype := reflect.TypeOf(tc.testStruct)
			sf := reflect.StructField{Name: "ConfigField", Type: itype}
			configStructType := reflect.StructOf([]reflect.StructField{sf})

			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())
			f := &FlattenMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, f)
			val, err := tfmr.Translate()
			require.NoError(t, err)

			// populate the flattened struct
			tc.modify(val)

			revVal, err := tfmr.ReverseTranslate(val)
			require.NoError(t, err)

			// check the returned value of the struct matches what is expected
			rv := revVal.FieldByName("ConfigField")
			tc.assertion(rv.Interface())
		})
	}
}
