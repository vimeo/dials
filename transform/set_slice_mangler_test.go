package transform

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestSetSliceManglerMangle(t *testing.T) {
	m := SetSliceMangler{}
	sf := reflect.StructField{
		Type: reflect.TypeOf(map[string]struct{}{}),
	}
	sfs, err := m.Mangle(sf)

	require.NoError(t, err)

	assert.Equal(t, reflect.TypeFor[[]string](), sfs[0].Type)
}

func TestSetSliceManglerUnmangle(t *testing.T) {
	cases := map[string]struct {
		StructFieldType reflect.Type
		SrcValue        any
		ExpectedMap     any
	}{
		"stringSet": {
			StructFieldType: reflect.TypeOf(map[string]struct{}{}),
			SrcValue:        []string{"John", "Paul", "George", "Ringo"},
			ExpectedMap: map[string]struct{}{
				"John":   {},
				"Paul":   {},
				"George": {},
				"Ringo":  {},
			},
		},
		"intSet": {
			StructFieldType: reflect.TypeOf(map[int]struct{}{}),
			SrcValue:        []int{1, 2, 3},
			ExpectedMap: map[int]struct{}{
				1: {},
				2: {},
				3: {},
			},
		},
		"setWithDuplicates": {
			StructFieldType: reflect.TypeOf(map[string]struct{}{}),
			SrcValue:        []string{"foo", "foo", "bar", "baz"},
			ExpectedMap: map[string]struct{}{
				"foo": {},
				"bar": {},
				"baz": {},
			},
		},
		"nil": {
			StructFieldType: reflect.TypeOf(map[int]struct{}{}),
			SrcValue:        []int(nil),
			ExpectedMap:     map[int]struct{}(nil),
		},
	}

	for name, c := range cases {
		testCase := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sf := reflect.StructField{Name: "ConfigField", Type: testCase.StructFieldType}
			configStructType := reflect.StructOf([]reflect.StructField{sf})
			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())

			m := &SetSliceMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, m)

			val, err := tfmr.Translate()
			require.NoError(t, err)

			val.Field(0).Set(reflect.ValueOf(testCase.SrcValue))

			unmangledVal, err := tfmr.ReverseTranslate(val)
			require.NoError(t, err)

			assert.Equal(t, testCase.ExpectedMap, unmangledVal.FieldByName("ConfigField").Interface())
		})
	}
}
