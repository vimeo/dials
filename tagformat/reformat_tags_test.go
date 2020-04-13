package tagformat

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/tagformat/caseconversion"
)

func TestReformatdialsTag(t *testing.T) {
	trm := NewTagReformattingMangler(DialsTagName, caseconversion.DecodeLowerCamelCase, caseconversion.EncodeLowerSnakeCase)
	sf := reflect.StructField{
		Tag: `dials:"testTag"`,
	}
	newSFSlice, err := trm.Mangle(sf)
	require.Len(t, newSFSlice, 1)
	require.NoError(t, err)
	assert.Equal(t, `dials:"test_tag"`, string(newSFSlice[0].Tag))
}

func TestNoTag(t *testing.T) {
	trm := NewTagReformattingMangler(DialsTagName, caseconversion.DecodeLowerCamelCase, caseconversion.EncodeLowerSnakeCase)
	sf := reflect.StructField{}
	newSFSlice, err := trm.Mangle(sf)
	require.Len(t, newSFSlice, 1)
	require.NoError(t, err)
	assert.Equal(t, "", string(newSFSlice[0].Tag))
}
