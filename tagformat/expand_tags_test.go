package tagformat

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
	"github.com/vimeo/dials/transform"
)

func TestExpanddialsTag(t *testing.T) {
	mangler := TagCopyingMangler{SrcTag: DialsTagName, NewTag: "json"}
	sf := reflect.StructField{
		Tag: `dials:"test"`,
	}
	newSFs, mangleErr := mangler.Mangle(sf)
	require.NoError(t, mangleErr)
	require.Len(t, newSFs, 1)
	assert.Equal(t, `dials:"test" json:"test"`, string(newSFs[0].Tag))
}

func TestExpanddialsTags(t *testing.T) {
	type testConfig struct {
		DatabaseName    string `dials:"database_name"`
		DatabaseAddress string `dials:"database_address"`
	}

	tc := testConfig{}
	vc := reflect.ValueOf(tc)
	cfg := ptrify.Pointerify(vc.Type(), vc)

	mangler := TagCopyingMangler{SrcTag: DialsTagName, NewTag: "yaml"}
	tfm := transform.NewTransformer(cfg, &mangler)

	mangledVal, mangleErr := tfm.Translate()
	require.NoError(t, mangleErr)

	typeWithExpandedTags := mangledVal.Type()

	assert.Equal(t, reflect.Struct, typeWithExpandedTags.Kind())
	for z := 0; z < typeWithExpandedTags.NumField(); z++ {
		t.Logf("field: %v", typeWithExpandedTags.Field(z))
	}

	{
		sf, found := typeWithExpandedTags.FieldByName("DatabaseName")
		if !found {
			t.Fatalf("missing DatabaseName field; type: %s", typeWithExpandedTags)
		}
		t.Logf("DatabaseName tag val: %q", sf.Tag)
		assert.Equal(t, sf.Tag.Get("yaml"), "database_name")
	}

	{
		sf, found := typeWithExpandedTags.FieldByName("DatabaseAddress")
		if !found {
			t.Fatalf("missing DatabaseAddress field; type: %s", typeWithExpandedTags)
		}
		t.Logf("DatabaseAddress tag val: %q", sf.Tag)
		assert.Equal(t, sf.Tag.Get("yaml"), "database_address")
	}
}
