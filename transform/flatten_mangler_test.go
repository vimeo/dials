package transform

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

type tu struct {
	Text      string
	Unmarshal string
}

// need a concrete type that implements TextUnmarshaler
func (u tu) UnmarshalText(data []byte) error {
	return nil
}

func TestFlattenMangler(t *testing.T) {
	type Foo struct {
		Location    string `dials:"Location"`
		Coordinates int    `dials:"Coordinates"`
	}

	type bar struct {
		Name         string `dials:"Name"`
		Foobar       *Foo   `dials:"Foobar"`
		AnotherField int    `dials:"AnotherField"`
	}

	type embeddedFooBar struct {
		Name string `dials:"Name"`
		Foo
		AnotherField int `dials:"AnotherField"`
	}

	type embeddedFooBarTag struct {
		Name         string `dials:"Name"`
		Foo          `dials:"embeddedFoo"`
		AnotherField int `dials:"AnotherField"`
	}

	b := bar{
		Name: "test",
		Foobar: &Foo{
			Location:    "here",
			Coordinates: 64,
		},
		AnotherField: 42,
	}

	efg := embeddedFooBar{
		Name: "test",
		Foo: Foo{
			Location:    "here",
			Coordinates: 64,
		},
		AnotherField: 42,
	}

	efgt := embeddedFooBarTag{
		"test",
		Foo{
			Location:    "here",
			Coordinates: 64,
		},
		42,
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
				assert.EqualValues(t, "ConfigField", val.Type().Field(0).Tag.Get(DialsTagName))
				assert.EqualValues(t, "ConfigField", val.Type().Field(0).Tag.Get(dialsFieldPathTag))
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
				assert.EqualValues(t, "ConfigField", val.Type().Field(0).Tag.Get(DialsTagName))
				assert.EqualValues(t, "ConfigField", val.Type().Field(0).Tag.Get(dialsFieldPathTag))

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
			name:       "one member in struct of type time.Time",
			testStruct: time.Time{},
			modify: func(t testing.TB, val reflect.Value) {
				assert.Equal(t, "ConfigField", val.Type().Field(0).Name)
				assert.Equal(t, "ConfigField", val.Type().Field(0).Tag.Get(DialsTagName))
				assert.Equal(t, "ConfigField", val.Type().Field(0).Tag.Get(dialsFieldPathTag))
				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				val.Field(0).Set(reflect.ValueOf(&curTime))
			},
			assertion: func(t testing.TB, i interface{}) {
				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				assert.EqualValues(t, curTime, *i.(*time.Time))
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

				expectedDialsTags := []string{
					"ConfigField_TestInt",
					"ConfigField_TestString",
					"ConfigField_TestBool",
				}

				expectedPathTags := []string{
					"ConfigField,TestInt",
					"ConfigField,TestString",
					"ConfigField,TestBool",
				}

				for i := 0; i < val.Type().NumField(); i++ {
					assert.EqualValues(t, expectedDialsTags[i], val.Type().Field(i).Tag.Get(DialsTagName))
					assert.EqualValues(t, expectedPathTags[i], val.Type().Field(i).Tag.Get(dialsFieldPathTag))
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

				expectedDialsTags := []string{
					"ConfigField_Name",
					"ConfigField_Foobar_Location",
					"ConfigField_Foobar_Coordinates",
					"ConfigField_AnotherField",
				}

				expectedFieldTags := []string{
					"ConfigField,Name",
					"ConfigField,Foobar,Location",
					"ConfigField,Foobar,Coordinates",
					"ConfigField,AnotherField",
				}

				for i := 0; i < val.Type().NumField(); i++ {
					assert.EqualValues(t, expectedDialsTags[i], val.Type().Field(i).Tag.Get(DialsTagName))
					assert.EqualValues(t, expectedFieldTags[i], val.Type().Field(i).Tag.Get(dialsFieldPathTag))

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

				assert.EqualValues(t, &b, i)
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
					"ConfigField_hello_jude",
					"ConfigField_here_comes_THE_sun",
					"ConfigField_YESTERDAY_Hello",
					"ConfigField_YESTERDAY_GoodBye_Penny",
					"ConfigField_YESTERDAY_GoodBye_Lane",
					"ConfigField_DayTripper",
				}

				expectedFieldPathTag := []string{
					"ConfigField,HeyJude",
					"ConfigField,ComeTogether",
					"ConfigField,Blackbird,Hello",
					"ConfigField,Blackbird,GoodBye,Penny",
					"ConfigField,Blackbird,GoodBye,Lane",
					"ConfigField,DayTripper",
				}

				for i := 0; i < len(expectedTags); i++ {
					assert.EqualValues(t, expectedTags[i], val.Type().Field(i).Tag.Get(DialsTagName))
					assert.EqualValues(t, expectedFieldPathTag[i], val.Type().Field(i).Tag.Get(dialsFieldPathTag))

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

				assert.EqualValues(t, &s, i)
			},
		},
		{
			name:       "Embedded struct without tag",
			testStruct: efg,
			modify: func(t testing.TB, val reflect.Value) {
				expectedDialsTags := []string{
					"ConfigField_Name",
					"ConfigField_Location",
					"ConfigField_Coordinates",
					"ConfigField_AnotherField",
				}

				expectedFieldTags := []string{
					"ConfigField,Name",
					"ConfigField,Foo,Location",
					"ConfigField,Foo,Coordinates",
					"ConfigField,AnotherField",
				}

				expectedNames := []string{
					"ConfigFieldName",
					"ConfigFieldLocation",
					"ConfigFieldCoordinates",
					"ConfigFieldAnotherField",
				}

				vtype := val.Type()
				for i := 0; i < vtype.NumField(); i++ {
					assert.EqualValues(t, expectedDialsTags[i], vtype.Field(i).Tag.Get(DialsTagName))
					assert.EqualValues(t, expectedFieldTags[i], vtype.Field(i).Tag.Get(dialsFieldPathTag))
					assert.EqualValues(t, expectedNames[i], vtype.Field(i).Name)
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
				// embedded fields are hard to compare with defined structs because
				// they are named but the Anonymous field is set to true. So use
				// JSON marshaling/unmarshalling to compare values

				b, err := json.Marshal(i)
				require.NoError(t, err)

				var actual embeddedFooBar
				err = json.Unmarshal(b, &actual)
				assert.NoError(t, err)
				assert.Equal(t, efg, actual)
			},
		},
		{
			name:       "Embedded struct with tag",
			testStruct: efgt,
			modify: func(t testing.TB, val reflect.Value) {
				expectedDialsTags := []string{
					"ConfigField_Name",
					"ConfigField_embeddedFoo_Location",
					"ConfigField_embeddedFoo_Coordinates",
					"ConfigField_AnotherField",
				}

				expectedFieldTags := []string{
					"ConfigField,Name",
					"ConfigField,Foo,Location",
					"ConfigField,Foo,Coordinates",
					"ConfigField,AnotherField",
				}

				expectedNames := []string{
					"ConfigFieldName",
					"ConfigFieldLocation",
					"ConfigFieldCoordinates",
					"ConfigFieldAnotherField",
				}

				vtype := val.Type()
				for i := 0; i < vtype.NumField(); i++ {
					assert.EqualValues(t, expectedDialsTags[i], vtype.Field(i).Tag.Get(DialsTagName))
					assert.EqualValues(t, expectedFieldTags[i], vtype.Field(i).Tag.Get(dialsFieldPathTag))
					assert.EqualValues(t, expectedNames[i], vtype.Field(i).Name)
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
				// assert.EqualValues doesn't work here with the embedded structs
				// like it does for nested structs since the values are different
				// with Anonymous set to true for embedded fields. So using JSON
				// marshalling to ensure that the values are populated correctly
				b, err := json.Marshal(i)
				require.NoError(t, err)

				var actual embeddedFooBarTag
				err = json.Unmarshal(b, &actual)
				assert.NoError(t, err)

				assert.Equal(t, efgt, actual)
			},
		},
		{
			name: "support time.Time",
			testStruct: &struct {
				A time.Time
				B int
			}{
				A: time.Time{},
				B: 8,
			},
			modify: func(t testing.TB, val reflect.Value) {
				require.Equal(t, 2, val.Type().NumField())

				assert.Equal(t, "ConfigFieldA", val.Type().Field(0).Name)
				assert.Equal(t, "ConfigFieldB", val.Type().Field(1).Name)

				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				int1 := 1
				val.Field(0).Set(reflect.ValueOf(&curTime))
				val.Field(1).Set(reflect.ValueOf(&int1))
			},
			assertion: func(t testing.TB, i interface{}) {
				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				int1 := 1

				b := &struct {
					A *time.Time `dials:"A"`
					B *int       `dials:"B"`
				}{
					A: &curTime,
					B: &int1,
				}
				assert.EqualValues(t, b, i)
			},
		},
		{
			name: "unmarshal text concrete type",
			testStruct: &struct {
				A time.Time
				B int
				T tu
			}{
				A: time.Time{},
				B: 8,
				T: tu{Text: "Hello", Unmarshal: "World"},
			},
			modify: func(t testing.TB, val reflect.Value) {
				require.Equal(t, 3, val.Type().NumField())

				s := []string{
					"ConfigFieldA",
					"ConfigFieldB",
					"ConfigFieldT",
				}

				for i := 0; i < val.Type().NumField(); i++ {
					assert.Equal(t, s[i], val.Type().Field(i).Name)
				}

				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				int1 := 1
				testtu := tu{
					Text:      "Hey",
					Unmarshal: "Jude",
				}
				val.Field(0).Set(reflect.ValueOf(&curTime))
				val.Field(1).Set(reflect.ValueOf(&int1))
				val.Field(2).Set(reflect.ValueOf(&testtu))
			},
			assertion: func(t testing.TB, i interface{}) {
				curTime, timeErr := time.Parse(time.Stamp, "May 18 15:04:05")
				require.NoError(t, timeErr)
				int1 := 1
				testtu := tu{
					Text:      "Hey",
					Unmarshal: "Jude",
				}

				b := &struct {
					A *time.Time `dials:"A"`
					B *int       `dials:"B"`
					T *tu        `dials:"T"`
				}{
					A: &curTime,
					B: &int1,
					T: &testtu,
				}
				assert.EqualValues(t, b, i)
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

type Embed struct {
	Foo string `dials:"foofoo"`
	Bar bool   // will have dials tag "Bar" after flatten mangler
}

func TestTopLevelEmbed(t *testing.T) {
	t.Parallel()
	type Config struct {
		unexposedHello string
		Hello          string
		Embed          `dials:"creative_name"`
	}

	c := &Config{
		unexposedHello: "hello world",
		Embed: Embed{
			Foo: "DoesThisWork",
		},
	}
	typeOfC := reflect.TypeOf(c)
	tVal := reflect.ValueOf(c)
	typeInstance := ptrify.Pointerify(typeOfC.Elem(), tVal.Elem())

	f := DefaultFlattenMangler()
	tfmr := NewTransformer(typeInstance, f)
	val, err := tfmr.Translate()
	require.NoError(t, err)

	expectedNames := []string{
		"Hello", "Foo", "Bar",
	}

	expectedDialsTags := []string{
		"Hello",
		"creative_name_foofoo",
		"creative_name_Bar",
	}

	expectedFieldTags := []string{
		"Hello", "Embed,Foo", "Embed,Bar",
	}

	for i := 0; i < val.Type().NumField(); i++ {
		assert.Equal(t, expectedNames[i], val.Type().Field(i).Name)
		assert.EqualValues(t, expectedDialsTags[i], val.Type().Field(i).Tag.Get(DialsTagName))
		assert.EqualValues(t, expectedFieldTags[i], val.Type().Field(i).Tag.Get(dialsFieldPathTag))
	}
}

func TestGetField(t *testing.T) {
	// used for pointers in tests
	pbool := true
	pint := 8
	pstring := "creative word"

	testcases := []struct {
		name       string
		testStruct interface{}
		expected   []interface{}
	}{
		{
			name: "zero-valued struct",
			testStruct: &struct {
				Hello   string
				Goodbye bool
			}{},
			expected: []interface{}{"", false},
		},
		{
			name: "simple_struct",
			testStruct: &struct {
				Hello       string
				littleHello string
				Goodbye     bool
			}{
				Hello:   "HeyJude",
				Goodbye: true,
			},
			// only two values in the array because the unexposed field
			// won't be iterated in reflect.Type.NumField()
			expected: []interface{}{"HeyJude", true},
		},
		{
			name: "pointerified fields",
			testStruct: &struct {
				Hello   *string
				Goodbye *bool
			}{
				Hello:   &pstring,
				Goodbye: &pbool,
			},
			expected: []interface {
			}{pstring, pbool},
		},
		{
			name: "empty_pointerified fields",
			testStruct: &struct {
				Hello   string
				Goodbye *bool
			}{},
			expected: []interface{}{"", false},
		},
		{
			name: "nested_struct",
			testStruct: &struct {
				Hello   string
				Goodbye struct {
					Here   bool
					Comes  *int
					TheSun string
				}
			}{
				Goodbye: struct {
					Here   bool
					Comes  *int
					TheSun string
				}{
					Here:   true,
					Comes:  &pint,
					TheSun: "not the moon",
				},
			},
			expected: []interface{}{"", true, pint, "not the moon"},
		},
		{
			name: "nested_empty_pointer_struct",
			testStruct: &struct {
				Hello   string
				Goodbye *struct {
					Here   bool
					Comes  *int
					TheSun string
				}
			}{},
			expected: []interface{}{"", false, 0, ""},
		},
		{
			name: "nested_struct_with_embedded_fields",
			testStruct: &struct {
				Hello string
				*Embed
			}{
				Embed: &Embed{
					Foo: "Foobars",
				},
			},
			expected: []interface{}{"", "Foobars", false},
		},
	}
	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cType := reflect.TypeOf(tc.testStruct)
			cVal := reflect.ValueOf(tc.testStruct)

			typeInstance := ptrify.Pointerify(cType.Elem(), cVal.Elem())

			f := DefaultFlattenMangler()
			tfmr := NewTransformer(typeInstance, f)
			val, err := tfmr.Translate()
			require.NoError(t, err)

			for i := 0; i < val.Type().NumField(); i++ {
				sf := val.Type().Field(i)
				assert.EqualValues(t, tc.expected[i], GetField(sf, cVal).Interface())
			}
		})
	}
}
