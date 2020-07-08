package env

import (
	"fmt"
	"os"
	"reflect"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/transform"
)

const envTagName = "dials_env"

// Source implements the dials.Source interface to set configuration from
// environment variables.
type Source struct {
	Prefix string
}

// Value fills in the user-provided config struct using environment variables.
// It looks up the environment variable to read into a given struct field by
// using that field's `dials_env` struct tag if present, then its `dials` tag if
// present, and finally its name. If the struct field's name is used, Value
// assumes the name is in Go-style camelCase (e.g., "JSONFilePath") and converts
// it to UPPER_SNAKE_CASE. (The casing of `dials_env` and `dials` tags is left
// unchanged.)
func (e *Source) Value(t *dials.Type) (reflect.Value, error) {
	// flatten the nested fields
	flattenMangler := transform.NewFlattenMangler(common.DialsTagName, caseconversion.EncodeUpperCamelCase, caseconversion.EncodeUpperCamelCase)
	// reformat the tags so they are SCREAMING_SNAKE_CASE
	reformatTagMangler := tagformat.NewTagReformattingMangler(common.DialsTagName, caseconversion.DecodeGoTags, caseconversion.EncodeUpperSnakeCase)
	// copy tags from "dials" to "dials_env" tag
	tagCopyingMangler := &tagformat.TagCopyingMangler{SrcTag: common.DialsTagName, NewTag: envTagName}
	// convert all the fields in the flattened struct to string type so the environment variables can be set
	stringCastingMangler := &transform.StringCastingMangler{}
	tfmr := transform.NewTransformer(t.Type(), flattenMangler, reformatTagMangler, tagCopyingMangler, stringCastingMangler)

	val, err := tfmr.Translate()
	if err != nil {
		return reflect.Value{}, err
	}

	valType := val.Type()
	for i := 0; i < val.NumField(); i++ {
		sf := valType.Field(i)
		envTagVal := sf.Tag.Get(envTagName)
		if envTagVal == "" {
			// dials_env tag should be populated because dials tag is populated
			// after flatten mangler and we copy from dials to dials_env tag
			panic(fmt.Errorf("Empty dials_env tag for field name %s", sf.Name))
		}

		if e.Prefix != "" {
			envTagVal = e.Prefix + "_" + envTagVal
		}

		if envVarVal, ok := os.LookupEnv(envTagVal); ok {
			// The StringCastingMangler has transformed all the fields on the
			// dials.Type into *string types, so that they can be set here as
			// strings (and when ReverseTranslate is called, cast into the
			// original types on the StructFields.)
			val.Field(i).Set(reflect.ValueOf(&envVarVal))
		}
	}

	return tfmr.ReverseTranslate(val)
}
