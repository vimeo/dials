package transform

import (
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

type simpleTextUnmarshaler struct {
	Val string
}

func (stm *simpleTextUnmarshaler) UnmarshalText(data []byte) error {
	stm.Val = string(data)
	return nil
}

func TestTextUnmarshalerManglerUnmangle(t *testing.T) {
	cases := map[string]struct {
		StructFieldType reflect.Type
		StringValue     string
		AssertFunc      func(testing.TB, any)
		ExpectedErr     string
	}{
		"TextUnmarshaler": {
			StructFieldType: reflect.TypeFor[net.IP](),
			StringValue:     "10.0.0.1",
			AssertFunc: func(t testing.TB, i any) {
				assert.Equal(t, net.ParseIP("10.0.0.1"), i)
			},
		},
		"*TextUnmarshaler": {
			StructFieldType: reflect.TypeFor[*net.IP](),
			StringValue:     "10.0.0.2",
			AssertFunc: func(t testing.TB, i any) {
				assert.Equal(t, net.ParseIP("10.0.0.2"), *(i.(*net.IP)))
			},
		},
		"TextUnmarshalerNil": {
			StructFieldType: reflect.TypeFor[net.IP](),
			StringValue:     "",
			AssertFunc: func(t testing.TB, i any) {
				var ip net.IP
				assert.Equal(t, ip, i)
			},
		},
		"CustomStructType": {
			StructFieldType: reflect.TypeFor[simpleTextUnmarshaler](),
			StringValue:     "foo",
			AssertFunc: func(t testing.TB, i any) {
				stm, ok := i.(*simpleTextUnmarshaler)
				require.True(t, ok)
				assert.Equal(t, "foo", stm.Val)
			},
		},
		"NotTextUnmarshaler": {
			StructFieldType: reflect.TypeFor[map[string]any](),
			StringValue:     "",
			AssertFunc: func(t testing.TB, i any) {
				var m map[string]any
				assert.Equal(t, m, i)
			},
		},
	}

	for name, c := range cases {
		testCase := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sf := reflect.StructField{Name: "ConfigField", Type: testCase.StructFieldType}
			configStructType := reflect.StructOf([]reflect.StructField{sf})
			ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())

			m := &TextUnmarshalerMangler{}
			tfmr := NewTransformer(ptrifiedConfigType, m)

			val, err := tfmr.Translate()
			require.NoError(t, err)

			if len(testCase.StringValue) > 0 {
				val.Field(0).Set(reflect.ValueOf(&testCase.StringValue))
			}

			unmangledVal, err := tfmr.ReverseTranslate(val)
			if testCase.ExpectedErr != "" {
				require.Contains(t, err.Error(), testCase.ExpectedErr)
				return
			}

			require.NoError(t, err)
			testCase.AssertFunc(t, unmangledVal.FieldByName("ConfigField").Interface())
		})
	}
}
