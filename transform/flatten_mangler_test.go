package transform

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestMangler(t *testing.T) {
	// TODO: figure out what to do with pointers for non structs (don't use the elem fieldType)
	type foo struct {
		Location string
	}

	type bar struct {
		Name   string
		Foobar *foo
	}

	b := bar{
		Name: "test",
		Foobar: &foo{
			Location: "here",
		},
	}

	testCases := []struct {
		name       string
		testStruct interface{}
		// modify will fill the value after the
		modify    func(reflect.Value)
		assertion func(interface{})
	}{
		{
			name:       "basic one layer struct",
			testStruct: 32,
			modify: func(val reflect.Value) {
				i := 32
				val.Field(0).Set(reflect.ValueOf(&i))
			},
			assertion: func(i interface{}) {
				fmt.Println("interface", i)
				assert.Equal(t, 32, *i.(*int))
			},
		},
		{
			name:       "nested struct",
			testStruct: b,
			modify: func(val reflect.Value) {
				s1 := "test"
				s2 := "here"

				val.Field(0).Set(reflect.ValueOf(&s1))
				val.Field(1).Set(reflect.ValueOf(&s2))
			},
			assertion: func(i interface{}) {
				s1 := "test"
				s2 := "here"
				b := struct {
					Name   *string
					Foobar *struct {
						Location *string
					}
				}{
					Name: &s1,
					Foobar: &struct {
						Location *string
					}{
						Location: &s2,
					},
				}

				assert.Equal(t, &b, i)
			},
		},
	}

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			itype := reflect.TypeOf(tc.testStruct)
			sf := reflect.StructField{Name: "ConfigField", Type: itype}
			configStructType := reflect.StructOf([]reflect.StructField{sf})

			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())
			f := &FlattenMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, f)
			val, err := tfmr.Translate()
			require.NoError(t, err)

			fmt.Printf("Mangled %+v %s \n", val.Interface(), val.String())

			tc.modify(val)

			fmt.Printf("Modified val %+v \n", val.Interface())

			revVal, err := tfmr.ReverseTranslate(val)
			require.NoError(t, err)
			fmt.Printf("Reverse Translated Val %+v %s\n", revVal, revVal.String())

			rv := revVal.FieldByName("ConfigField")
			fmt.Printf("rv %+v %s", rv, rv.String())
			fmt.Printf("rv.Interface() %+v %s", rv.Interface(), rv.String())
			tc.assertion(rv.Interface())

		})
	}
}
