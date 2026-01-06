package transform

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials/parse"
	"github.com/vimeo/dials/ptrify"
)

func TestStringCastingManglerMangle(t *testing.T) {
	m := StringCastingMangler{}
	sf := reflect.StructField{
		Type: reflect.TypeFor[int](),
	}
	sfs, err := m.Mangle(sf)

	require.NoError(t, err)

	assert.Equal(t, strPtrType, sfs[0].Type)
}

func TestStringCastingManglerUnmangle(t *testing.T) {
	cases := map[string]struct {
		StructFieldType reflect.Type
		StringValue     string
		AssertFunc      func(any)
		ExpectedErr     string
	}{
		"string": {
			StructFieldType: reflect.TypeFor[string](),
			StringValue:     "asdf",
			AssertFunc: func(i any) {
				assert.Equal(t, "asdf", *(i.(*string)))
			},
		},
		"bool": {
			StructFieldType: reflect.TypeFor[bool](),
			StringValue:     "true",
			AssertFunc: func(i any) {
				assert.Equal(t, true, *(i.(*bool)))
			},
		},
		"int": {
			StructFieldType: reflect.TypeFor[int](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, 1, *(i.(*int)))
			},
		},
		"int8": {
			StructFieldType: reflect.TypeFor[int8](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, int8(1), *(i.(*int8)))
			},
		},
		"int16": {
			StructFieldType: reflect.TypeFor[int16](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, int16(1), *(i.(*int16)))
			},
		},
		"int32": {
			StructFieldType: reflect.TypeFor[int32](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, int32(1), *(i.(*int32)))
			},
		},
		"int64": {
			StructFieldType: reflect.TypeFor[int64](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, int64(1), *(i.(*int64)))
			},
		},
		"uint": {
			StructFieldType: reflect.TypeFor[uint](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, uint(1), *(i.(*uint)))
			},
		},
		"uint8": {
			StructFieldType: reflect.TypeFor[uint8](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, uint8(1), *(i.(*uint8)))
			},
		},
		"uint16": {
			StructFieldType: reflect.TypeFor[uint16](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, uint16(1), *(i.(*uint16)))
			},
		},
		"uint32": {
			StructFieldType: reflect.TypeFor[uint32](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, uint32(1), *(i.(*uint32)))
			},
		},
		"uint64": {
			StructFieldType: reflect.TypeFor[uint64](),
			StringValue:     "1",
			AssertFunc: func(i any) {
				assert.Equal(t, uint64(1), *(i.(*uint64)))
			},
		},
		"float32": {
			StructFieldType: reflect.TypeFor[float32](),
			StringValue:     "1.5",
			AssertFunc: func(i any) {
				assert.Equal(t, float32(1.5), *(i.(*float32)))
			},
		},
		"float64": {
			StructFieldType: reflect.TypeFor[float64](),
			StringValue:     "1.9",
			AssertFunc: func(i any) {
				assert.Equal(t, 1.9, *(i.(*float64)))
			},
		},
		"complex64": {
			StructFieldType: reflect.TypeFor[complex64](),
			StringValue:     "10+3i",
			AssertFunc: func(i any) {
				assert.Equal(t, complex64(10+3i), *(i.(*complex64)))
			},
		},
		"complex128": {
			StructFieldType: reflect.TypeFor[complex128](),
			StringValue:     "10+3i",
			AssertFunc: func(i any) {
				assert.Equal(t, complex128(10+3i), *(i.(*complex128)))
			},
		},
		"duration": {
			StructFieldType: reflect.TypeFor[time.Duration](),
			StringValue:     "1h",
			AssertFunc: func(i any) {
				assert.Equal(t, time.Duration(3600000000000), *(i.(*time.Duration)))
			},
		},
		"duration_error": {
			StructFieldType: reflect.TypeFor[time.Duration](),
			StringValue:     "1",
			ExpectedErr:     "missing unit in duration ",
		},
		"string_slice": {
			StructFieldType: reflect.TypeFor[[]string](),
			StringValue:     `a,b,c`,
			AssertFunc: func(i any) {
				expected := []string{"a", "b", "c"}
				actual := i.([]string)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"int_slice": {
			StructFieldType: reflect.TypeFor[[]int](),
			StringValue:     `1,2,3`,
			AssertFunc: func(i any) {
				expected := []int{1, 2, 3}
				actual := i.([]int)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"int8_slice": {
			StructFieldType: reflect.TypeFor[[]int8](),
			StringValue:     `1,2,3`,
			AssertFunc: func(i any) {
				expected := []int8{1, 2, 3}
				actual := i.([]int8)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"float64_slice": {
			StructFieldType: reflect.TypeFor[[]float64](),
			StringValue:     `1.1, 2.1, 3.1`,
			AssertFunc: func(i any) {
				expected := []float64{1.1, 2.1, 3.1}
				actual := i.([]float64)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"complex128_slice": {
			StructFieldType: reflect.TypeFor[[]complex128](),
			StringValue:     `"10+3i", "5+2i", "3+3i"`,
			AssertFunc: func(i any) {
				expected := []complex128{10 + 3i, 5 + 2i, 3 + 3i}
				actual := i.([]complex128)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"string_string_map": {
			StructFieldType: reflect.TypeFor[map[string]string](),
			StringValue:     `"Origin": "foobar", "Referer": "fimbat"`,
			AssertFunc: func(i any) {
				expected := map[string]string{
					"Origin":  "foobar",
					"Referer": "fimbat",
				}
				actual := i.(map[string]string)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"string_string_slice_map": {
			StructFieldType: reflect.TypeFor[map[string][]string](),
			StringValue:     `"Origin": "foobar", "Origin": "foobat", "Referer": "fimbat"`,
			AssertFunc: func(i any) {
				expected := map[string][]string{
					"Origin":  {"foobar", "foobat"},
					"Referer": {"fimbat"},
				}
				actual := i.(map[string][]string)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"int_int_map": {
			StructFieldType: reflect.TypeFor[map[int]int](),
			StringValue:     `1: 1, 2: 3, 10: 8`,
			AssertFunc: func(i any) {
				expected := map[int]int{
					1:  1,
					2:  3,
					10: 8,
				}
				actual := i.(map[int]int)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"int_str_map": {
			StructFieldType: reflect.TypeFor[map[int]string](),
			StringValue:     `1: "a", 2: "b", 10: "c"`,
			AssertFunc: func(i any) {
				expected := map[int]string{
					1:  "a",
					2:  "b",
					10: "c",
				}
				actual := i.(map[int]string)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"str_int_map": {
			StructFieldType: reflect.TypeFor[map[string]int](),
			StringValue:     `"a": 1, "b": 2, "c": 10`,
			AssertFunc: func(i any) {
				expected := map[string]int{
					"a": 1,
					"b": 2,
					"c": 10,
				}
				actual := i.(map[string]int)
				assert.True(t, reflect.DeepEqual(expected, actual))
				// t.FailNow()
			},
		},
		"str_bool_map": {
			StructFieldType: reflect.TypeFor[map[string]bool](),
			StringValue:     `"a": true, "b": false, "c": true`,
			AssertFunc: func(i any) {
				expected := map[string]bool{
					"a": true,
					"b": false,
					"c": true,
				}
				actual := i.(map[string]bool)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"str_complex_map": {
			StructFieldType: reflect.TypeFor[map[string]complex128](),
			StringValue:     `"asdf": "3+5i", "b": "3+5i", "c": "3+5i"`,
			AssertFunc: func(i any) {
				expected := map[string]complex128{
					"asdf": complex128(3 + 5i),
					"b":    complex128(3 + 5i),
					"c":    complex128(3 + 5i),
				}
				actual := i.(map[string]complex128)
				assert.EqualValues(t, expected, actual)
			},
		},
		"complex_bool_map": {
			StructFieldType: reflect.TypeFor[map[complex64]bool](),
			StringValue:     `"3+5i": true, "10+5i": false, "1+2i": true`,
			AssertFunc: func(i any) {
				expected := map[complex64]bool{
					complex64(3 + 5i):  true,
					complex64(10 + 5i): false,
					complex64(1 + 2i):  true,
				}
				actual := i.(map[complex64]bool)
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
		"invalid_map": {
			StructFieldType: reflect.TypeFor[map[string][]int](),
			StringValue:     `"asdf": 1, "asdf": 2, "zxcv": 3`,
			ExpectedErr:     "unsupported map type",
		},
		"string_set": {
			StructFieldType: reflect.TypeOf(map[string]struct{}{}),
			StringValue:     `"a", "b"`,
			AssertFunc: func(i any) {
				expected := map[string]struct{}{
					"a": {},
					"b": {},
				}
				actual := i.(map[string]struct{})
				assert.True(t, reflect.DeepEqual(expected, actual))
			},
		},
	}

	for n, c := range cases {
		name := n
		testCase := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sf := reflect.StructField{Name: "ConfigField", Type: testCase.StructFieldType}
			configStructType := reflect.StructOf([]reflect.StructField{sf})
			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())

			m := &StringCastingMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, m)

			val, err := tfmr.Translate()
			require.NoError(t, err)

			nv := reflect.New(val.Field(0).Type())
			nv.Elem().Set(reflect.ValueOf(&testCase.StringValue))
			val.Field(0).Set(nv.Elem())

			unmangledVal, err := tfmr.ReverseTranslate(val)
			if testCase.ExpectedErr != "" {
				require.Contains(t, err.Error(), testCase.ExpectedErr)
				return
			}

			require.NoError(t, err)
			testCase.AssertFunc(unmangledVal.FieldByName("ConfigField").Interface())
		})
	}
}

func TestParseOverflow(t *testing.T) {
	cases := map[string]struct {
		StructFieldType reflect.Type
		StringValue     string
	}{
		"int8": {
			StructFieldType: reflect.TypeFor[int8](),
			StringValue:     "128",
		},
		"int16": {
			StructFieldType: reflect.TypeFor[int16](),
			StringValue:     "32768",
		},
		"int32": {
			StructFieldType: reflect.TypeFor[int32](),
			StringValue:     "2147483648",
		},
		"int64": {
			StructFieldType: reflect.TypeFor[int64](),
			StringValue:     "9223372036854775808",
		},
		"uint8": {
			StructFieldType: reflect.TypeFor[uint8](),
			StringValue:     "256",
		},
		"uint16": {
			StructFieldType: reflect.TypeFor[uint16](),
			StringValue:     "65537",
		},
		"uint32": {
			StructFieldType: reflect.TypeFor[uint32](),
			StringValue:     "4294967296",
		},
		"uint64": {
			StructFieldType: reflect.TypeFor[uint64](),
			StringValue:     "18446744073709551616",
		},
		"float32": {
			StructFieldType: reflect.TypeFor[float32](),
			StringValue:     "1e+40",
		},
		"float64": {
			StructFieldType: reflect.TypeFor[float64](),
			StringValue:     "1e+400",
		},
		"complex64": {
			StructFieldType: reflect.TypeFor[complex64](),
			StringValue:     "1e+400",
		},
		"complex128": {
			StructFieldType: reflect.TypeFor[complex64](),
			StringValue:     "1e+400",
		},
	}

	for n, c := range cases {
		name := n
		testCase := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sf := reflect.StructField{Name: "ConfigField", Type: testCase.StructFieldType}
			configStructType := reflect.StructOf([]reflect.StructField{sf})
			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())

			m := &StringCastingMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, m)

			val, translateErr := tfmr.Translate()
			if translateErr != nil {
				t.Fatal(translateErr)
			}

			nv := reflect.New(val.Field(0).Type())
			nv.Elem().Set(reflect.ValueOf(&testCase.StringValue))
			val.Field(0).Set(nv.Elem())

			_, err := tfmr.ReverseTranslate(val)
			if err != nil {
				reverseTranslateErr, isReverseTranslateErr := err.(*ReverseTranslateError)
				if !assert.True(t, isReverseTranslateErr) {
					t.Fatal()
				}

				unmangleErr, isUnmangleErr := reverseTranslateErr.Unwrap().(*UnmangleError)
				if !assert.True(t, isUnmangleErr) {
					t.Fatal()
				}

				_, isOverflowErr := unmangleErr.Unwrap().(*parse.OverflowError)
				_, isParseNumberErr := unmangleErr.Unwrap().(*parse.NumberError)
				if !assert.True(t, isOverflowErr || isParseNumberErr) {
					t.Fatal()
				}
			} else {
				t.Fatal("ReverseTranslate did not generate an error")
			}
		})
	}
}
