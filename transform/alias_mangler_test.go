package transform

import (
	"reflect"
	"testing"

	"github.com/fatih/structtag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasManglerMangle(t *testing.T) {

	for testName, itbl := range map[string]struct {
		tag           string
		expectedOrig  map[string]string
		expectedAlias map[string]string
	}{
		"dialsOnly": {
			tag:           `dials:"name" dialsalias:"anothername"`,
			expectedOrig:  map[string]string{"dials": "name"},
			expectedAlias: map[string]string{"dials": "anothername", "dialsdesc": "base dialsdesc unset (alias of dials=name)"},
		},
		"withDialsDesc": {
			tag:           `dials:"name" dialsalias:"anothername" dialsdesc:"the name for this"`,
			expectedOrig:  map[string]string{"dials": "name", "dialsdesc": "the name for this"},
			expectedAlias: map[string]string{"dials": "anothername", "dialsdesc": "the name for this (alias of dials=name)"},
		},
		"dialsDialsEnvFlagPFlag": {
			tag:           `dials:"name" dialsflag:"flagname" dialspflag:"pflagname" dialsenv:"envname" dialsalias:"anothername" dialsflagalias:"flagalias" dialspflagalias:"pflagalias" dialsenvalias:"envalias"`,
			expectedOrig:  map[string]string{"dials": "name", "dialsflag": "flagname", "dialspflag": "pflagname", "dialsenv": "envname"},
			expectedAlias: map[string]string{"dials": "anothername", "dialsflag": "flagalias", "dialspflag": "pflagalias", "dialsenv": "envalias", "dialsdesc": "base dialsdesc unset (alias of dials=name dialsenv=envname dialsflag=flagname dialspflag=pflagname)"},
		},
		"dialsDialsEnvWithDialsDesc": {
			tag:           `dials:"name" dialsenv:"thename" dialsalias:"anothername" dialsenvalias:"theothername, "dialsdesc:"THE description"`,
			expectedOrig:  map[string]string{"dials": "name", "dialsenv": "thename", "dialsdesc": "THE description"},
			expectedAlias: map[string]string{"dials": "anothername", "dialsenv": "theothername", "dialsdesc": "THE description (alias of dials=name dialsenv=thename)"},
		},
	} {
		tbl := itbl
		t.Run(testName, func(t *testing.T) {
			sf := reflect.StructField{
				Name: "Foo",
				Type: reflect.TypeFor[string](),
				Tag:  reflect.StructTag(tbl.tag),
			}

			aliasMangler := NewAliasMangler("dials", "dialsenv", "dialsflag", "dialspflag")
			fields, mangleErr := aliasMangler.Mangle(sf)
			require.NoError(t, mangleErr)

			require.Len(t, fields, 2)

			originalTags, parseErr := structtag.Parse(string(fields[0].Tag))
			require.NoError(t, parseErr)

			for k, v := range tbl.expectedOrig {
				val, err := originalTags.Get(k)
				require.NoError(t, err, "expected tag %s to be found", k)
				assert.Equal(t, v, val.Name)
			}
			assert.Equal(t, len(tbl.expectedOrig), originalTags.Len())

			aliasTags, parseErr := structtag.Parse(string(fields[1].Tag))
			require.NoError(t, parseErr)

			for k, v := range tbl.expectedAlias {
				val, err := aliasTags.Get(k)
				require.NoError(t, err)
				assert.Equal(t, v, val.Name)
			}
			assert.Equal(t, len(tbl.expectedAlias), aliasTags.Len())
		})
	}
}

func TestAliasManglerUnmangle(t *testing.T) {
	sf := reflect.StructField{
		Name: "Foo",
		Type: reflect.TypeFor[string](),
	}

	num := 42
	var nilInt *int

	aliasMangler := &AliasMangler{}

	originalSet := []FieldValueTuple{
		{
			Field: sf,
			Value: reflect.ValueOf(&num),
		},
		{
			Field: sf,
			Value: reflect.ValueOf(nilInt),
		},
	}

	val, err := aliasMangler.Unmangle(sf, originalSet)
	require.NoError(t, err)

	assert.Equal(t, 42, val.Elem().Interface())

	aliasSet := []FieldValueTuple{
		{
			Field: sf,
			Value: reflect.ValueOf(nilInt),
		},
		{
			Field: sf,
			Value: reflect.ValueOf(&num),
		},
	}

	val, err = aliasMangler.Unmangle(sf, aliasSet)
	require.NoError(t, err)

	assert.Equal(t, 42, val.Elem().Interface())

	bothSet := []FieldValueTuple{
		{
			Field: sf,
			Value: reflect.ValueOf(&num),
		},
		{
			Field: sf,
			Value: reflect.ValueOf(&num),
		},
	}

	_, err = aliasMangler.Unmangle(sf, bothSet)
	assert.NotNil(t, err) // there should be an error if both are set!

	neitherSet := []FieldValueTuple{
		{
			Field: sf,
			Value: reflect.ValueOf(nilInt),
		},
		{
			Field: sf,
			Value: reflect.ValueOf(nilInt),
		},
	}

	val, err = aliasMangler.Unmangle(sf, neitherSet)
	require.NoError(t, err)

	assert.True(t, val.IsNil())

	noAlias := []FieldValueTuple{
		{
			Field: sf,
			Value: reflect.ValueOf(&num),
		},
	}

	val, err = aliasMangler.Unmangle(sf, noAlias)
	require.NoError(t, err)

	assert.Equal(t, 42, val.Elem().Interface())
}
