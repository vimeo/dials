package tagformat

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/ptrify"
	"github.com/vimeo/dials/transform"
)

func TestExpandDialsTag(t *testing.T) {
	t.Parallel()
	mangler := TagCopyingMangler{SrcTag: common.DialsTagName, NewTag: "json"}
	sf := reflect.StructField{
		Tag: `dials:"test"`,
	}
	newSFs, mangleErr := mangler.Mangle(sf)
	require.NoError(t, mangleErr)
	require.Len(t, newSFs, 1)
	assert.Equal(t, `dials:"test" json:"test"`, string(newSFs[0].Tag))
}

func TestTagCopyingMangler(t *testing.T) {
	type inner struct {
		User string `dials:"user"`
	}

	type nested struct {
		YAMLConfig string  `dials:"config"`
		List       []inner `dials:"inners"`
	}

	testcases := []struct {
		name       string
		testStruct any
		tag        string
		assertion  func(t testing.TB, val reflect.Value, tagName string)
	}{
		{
			name: "one layered struct",
			tag:  "yaml",
			testStruct: struct {
				User     string `dials:"user"`
				Password string `dials:"password"`
			}{},
			assertion: func(t testing.TB, val reflect.Value, tagName string) {

				sf, ok := val.Type().FieldByName("User")
				require.True(t, ok)
				assert.Equal(t, "user", sf.Tag.Get(tagName))

				sf, ok = val.Type().FieldByName("Password")
				require.True(t, ok)
				assert.Equal(t, "password", sf.Tag.Get(tagName))
			},
		},
		{
			name: "nested struct",
			tag:  "yaml",
			testStruct: struct {
				DatabaseName    string `dials:"database_name"`
				DatabaseAddress string `dials:"database_address"`
				Nested          nested
			}{},
			assertion: func(t testing.TB, val reflect.Value, tagName string) {
				sf, ok := val.Type().FieldByName("DatabaseName")
				require.True(t, ok)
				assert.Equal(t, "database_name", sf.Tag.Get(tagName))

				sf, ok = val.Type().FieldByName("DatabaseAddress")
				require.True(t, ok)
				assert.Equal(t, "database_address", sf.Tag.Get(tagName))
			},
		},
		{
			name: "struct with slice of struct",
			tag:  "yaml",
			testStruct: struct {
				Vals []inner `dials:"the_vals"`
			}{},
			assertion: func(t testing.TB, val reflect.Value, tagName string) {
				sf, ok := val.Type().FieldByName("Vals")
				require.True(t, ok)
				assert.Equal(t, "the_vals", sf.Tag.Get(tagName))
			},
		},
	}

	for _, testCase := range testcases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vc := reflect.ValueOf(tc.testStruct)
			cfg := ptrify.Pointerify(vc.Type(), vc)
			mangler := TagCopyingMangler{SrcTag: common.DialsTagName, NewTag: tc.tag}
			tfm := transform.NewTransformer(cfg, &mangler)

			mangledVal, mangleErr := tfm.Translate()
			require.NoError(t, mangleErr)
			tc.assertion(t, mangledVal, tc.tag)

			_, revErr := tfm.ReverseTranslate(mangledVal)
			assert.NoError(t, revErr)
		})
	}
}
