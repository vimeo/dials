package transform

import (
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestTextUnmarshalerManglerUnmangle(t *testing.T) {
	cases := map[string]struct {
		StructFieldType reflect.Type
		StringValue     string
		AssertFunc      func(interface{})
		ExpectedErr     string
	}{
		"TextUnmarshaler": {
			StructFieldType: reflect.TypeOf(net.IP{}),
			StringValue:     "10.0.0.1",
			AssertFunc: func(i interface{}) {
				assert.Equal(t, net.ParseIP("10.0.0.1"), i)
			},
		},
		"*TextUnmarshaler": {
			StructFieldType: reflect.TypeOf(&net.IP{}),
			StringValue:     "10.0.0.2",
			AssertFunc: func(i interface{}) {
				assert.Equal(t, net.ParseIP("10.0.0.2"), *(i.(*net.IP)))
			},
		},
		"TextUnmarshalerNil": {
			StructFieldType: reflect.TypeOf(net.IP{}),
			StringValue:     "",
			AssertFunc: func(i interface{}) {
				var ip net.IP
				assert.Equal(t, ip, i)
			},
		},
		"NotTextUnmarshaler": {
			StructFieldType: reflect.TypeOf(map[string]interface{}{}),
			StringValue:     "",
			AssertFunc: func(i interface{}) {
				var m map[string]interface{}
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
			testCase.AssertFunc(unmangledVal.FieldByName("ConfigField").Interface())
		})
	}
}
