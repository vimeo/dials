package transform

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestFlattenMangler(t *testing.T) {
	type foo struct {
		Location    string `dials:"Location"`
		Coordinates int    `dials:"Coordinates"`
	}

	type bar struct {
		Name         string `dials:"Name"`
		Foobar       *foo   `dials:"Foobar"`
		AnotherField int    `dials:"AnotherField"`
	}

	b := bar{
		Name: "test",
		Foobar: &foo{
			Location:    "here",
			Coordinates: 64,
		},
		AnotherField: 42,
	}

	testCases := []struct {
		name       string
		testStruct interface{}
		// modify will fill the flatten struct value after Mangling
		modify    func(t testing.TB, val reflect.Value)
		assertion func(t testing.TB, i interface{})
	}{
		{
			name:       "one member in struct of type int",
			testStruct: 32,
			modify: func(t testing.TB, val reflect.Value) {
				assert.EqualValues(t, `dials:"ConfigField"`, val.Type().Field(0).Tag)
				i := 32
				val.Field(0).Set(reflect.ValueOf(&i))
			},
			assertion: func(t testing.TB, i interface{}) {
				assert.Equal(t, 32, *i.(*int))
			},
		},
		{
			name:       "one member in struct of type map",
			testStruct: map[string]string{},
			modify: func(t testing.TB, val reflect.Value) {
				assert.EqualValues(t, `dials:"ConfigField"`, val.Type().Field(0).Tag)
				m := map[string]string{
					"hello":   "world",
					"flatten": "unflatten",
				}
				val.Field(0).Set(reflect.ValueOf(m))
			},
			assertion: func(t testing.TB, i interface{}) {
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
			modify: func(t testing.TB, val reflect.Value) {},
			assertion: func(t testing.TB, i interface{}) {
				// should be empty struct since none of the fields are exposed
				assert.Equal(t, struct{}{}, *i.(*struct{}))
			},
		},
		{
			name: "one level nested struct exposed fields",
			testStruct: struct {
				TestInt    int    `dials:"TestInt"`
				TestString string `dials:"TestString"`
				TestBool   bool   `dials:"TestBool"`
			}{
				TestInt:    42,
				TestString: "hello world",
				TestBool:   true,
			},
			modify: func(t testing.TB, val reflect.Value) {

				expectedTags := []string{
					`dials:"ConfigField_TestInt"`,
					`dials:"ConfigField_TestString"`,
					`dials:"ConfigField_TestBool"`,
				}

				for i := 0; i < len(expectedTags); i++ {
					assert.EqualValues(t, expectedTags[i], val.Type().Field(i).Tag)
				}

				i := 42
				s := "hello world"
				b := true

				val.Field(0).Set(reflect.ValueOf(&i))
				val.Field(1).Set(reflect.ValueOf(&s))
				val.Field(2).Set(reflect.ValueOf(&b))
			},
			assertion: func(t testing.TB, i interface{}) {
				in := 42
				s := "hello world"
				b := true

				st := &struct {
					TestInt    *int    `dials:"TestInt"`
					TestString *string `dials:"TestString"`
					TestBool   *bool   `dials:"TestBool"`
				}{
					TestInt:    &in,
					TestString: &s,
					TestBool:   &b,
				}
				assert.Equal(t, st, i)
			},
		},
		{
			name:       "multilevel nested struct",
			testStruct: b,
			modify: func(t testing.TB, val reflect.Value) {

				expectedTags := []string{
					`dials:"ConfigField_Name"`,
					`dials:"ConfigField_Foobar_Location"`,
					`dials:"ConfigField_Foobar_Coordinates"`,
					`dials:"ConfigField_AnotherField"`,
				}

				for i := 0; i < len(expectedTags); i++ {
					assert.EqualValues(t, expectedTags[i], val.Type().Field(i).Tag)
				}

				s1 := "test"
				s2 := "here"
				i1 := 64
				i2 := 42

				val.Field(0).Set(reflect.ValueOf(&s1))
				val.Field(1).Set(reflect.ValueOf(&s2))
				val.Field(2).Set(reflect.ValueOf(&i1))
				val.Field(3).Set(reflect.ValueOf(&i2))
			},
			assertion: func(t testing.TB, i interface{}) {
				// all the fields are pointerified because of call to Pointerify
				s1 := "test"
				s2 := "here"
				i1 := 64
				i2 := 42
				b := struct {
					Name   *string `dials:"Name"`
					Foobar *struct {
						Location    *string `dials:"Location"`
						Coordinates *int    `dials:"Coordinates"`
					} `dials:"Foobar"`
					AnotherField *int `dials:"AnotherField"`
				}{
					Name: &s1,
					Foobar: &struct {
						Location    *string `dials:"Location"`
						Coordinates *int    `dials:"Coordinates"`
					}{
						Location:    &s2,
						Coordinates: &i1,
					},
					AnotherField: &i2,
				}
				assert.Equal(t, &b, i)
			},
		},
		{
			name: "multilevel nested with different struct tags",
			testStruct: struct {
				HeyJude      string `dials:"hello_jude"`
				ComeTogether int    `dials:"here_comes_THE_sun"`
				Blackbird    struct {
					Hello   int // doesn't have a tag on purpose
					GoodBye struct {
						Penny bool
						Lane  int64
					}
				} `dials:"YESTERDAY"`
				DayTripper bool
			}{},
			modify: func(t testing.TB, val reflect.Value) {
				expectedTags := []string{
					`dials:"ConfigField_hello_jude"`,
					`dials:"ConfigField_here_comes_THE_sun"`,
					`dials:"ConfigField_YESTERDAY_Hello"`,
					`dials:"ConfigField_YESTERDAY_GoodBye_Penny"`,
					`dials:"ConfigField_YESTERDAY_GoodBye_Lane"`,
					`dials:"ConfigField_DayTripper"`,
				}

				for i := 0; i < len(expectedTags); i++ {
					assert.EqualValues(t, expectedTags[i], val.Type().Field(i).Tag)
				}

				s1 := "The Beatles"
				i1 := 4
				i2 := 1900
				b1 := true
				i3 := int64(2020)
				b2 := false

				val.Field(0).Set(reflect.ValueOf(&s1))
				val.Field(1).Set(reflect.ValueOf(&i1))
				val.Field(2).Set(reflect.ValueOf(&i2))
				val.Field(3).Set(reflect.ValueOf(&b1))
				val.Field(4).Set(reflect.ValueOf(&i3))
				val.Field(5).Set(reflect.ValueOf(&b2))
			},
			assertion: func(t testing.TB, i interface{}) {
				s1 := "The Beatles"
				i1 := 4
				i2 := 1900
				b1 := true
				i3 := int64(2020)
				b2 := false

				s := struct {
					HeyJude      *string `dials:"hello_jude"`
					ComeTogether *int    `dials:"here_comes_THE_sun"`
					Blackbird    *struct {
						Hello   *int
						GoodBye *struct {
							Penny *bool
							Lane  *int64
						}
					} `dials:"YESTERDAY"`
					DayTripper *bool
				}{
					HeyJude:      &s1,
					ComeTogether: &i1,
					Blackbird: &struct {
						Hello   *int
						GoodBye *struct {
							Penny *bool
							Lane  *int64
						}
					}{
						Hello: &i2,
						GoodBye: &struct {
							Penny *bool
							Lane  *int64
						}{
							Penny: &b1,
							Lane:  &i3,
						},
					},
					DayTripper: &b2,
				}

				assert.Equal(t, &s, i)
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
			f := DefaultFlattenMangler()
			tfmr := NewTransformer(ptrifiedConfigType, f)
			val, err := tfmr.Translate()
			require.NoError(t, err)

			// populate the flattened struct
			tc.modify(t, val)

			revVal, err := tfmr.ReverseTranslate(val)
			require.NoError(t, err)

			// check the returned value of the struct matches what is expected
			rv := revVal.FieldByName("ConfigField")
			tc.assertion(t, rv.Interface())
		})
	}
}
