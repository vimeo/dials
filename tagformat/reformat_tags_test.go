package tagformat

import (
	"reflect"
	"testing"

	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/tagformat/caseconversion"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReformatdialsTag(t *testing.T) {
	trm := NewTagReformattingMangler(common.DialsTagName, caseconversion.DecodeLowerCamelCase, caseconversion.EncodeLowerSnakeCase)
	sf := reflect.StructField{
		Tag: `dials:"testTag"`,
	}
	newSFSlice, err := trm.Mangle(sf)
	require.Len(t, newSFSlice, 1)
	require.NoError(t, err)
	assert.Equal(t, `dials:"test_tag"`, string(newSFSlice[0].Tag))
}

func TestNoTag(t *testing.T) {
	trm := NewTagReformattingMangler(common.DialsTagName, caseconversion.DecodeGoCamelCase, caseconversion.EncodeLowerSnakeCase)
	sf := reflect.StructField{
		Name: "FieldName",
	}
	newSFSlice, err := trm.Mangle(sf)
	require.Len(t, newSFSlice, 1)
	require.NoError(t, err)
	assert.Equal(t, `dials:"field_name"`, string(newSFSlice[0].Tag))
}
