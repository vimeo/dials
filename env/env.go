package env

import (
	"fmt"
	"os"
	"reflect"

	"github.com/vimeo/dials"
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
	// copy tags from "dials" to "dials_env" tag
	tagCopyingMangler := &tagformat.TagCopyingMangler{SrcTag: transform.DialsTagName, NewTag: envTagName}
	// flatten the nested fields
	flattenMangler := transform.NewFlattenMangler(transform.DialsTagName, caseconversion.EncodeUpperCamelCase, caseconversion.EncodeUpperCamelCase)
	// reformat the tags so they are SCREAMING_SNAKE_CASE
	reformatTagMangler := tagformat.NewTagReformattingMangler(transform.DialsTagName, caseconversion.DecodeGolangCamelCase, caseconversion.EncodeUpperSnakeCase)
	// convert all the fields in the flattened struct to string type the environment variables can be set
	stringCastingMangler := &transform.StringCastingMangler{}
	tfmr := transform.NewTransformer(t.Type(), tagCopyingMangler, flattenMangler, reformatTagMangler, stringCastingMangler)

	val, err := tfmr.Translate()
	if err != nil {
		return reflect.Value{}, err
	}

	valType := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := valType.Field(i)
		envVarName := envVarName(e.Prefix, field)

		if envVarVal, ok := os.LookupEnv(envVarName); ok {
			// The StringCastingMangler has transformed all the fields on the
			// dials.Type into *string types, so that they can be set here as
			// strings (and when ReverseTranslate is called, cast into the
			// original types on the StructFields.)
			val.Field(i).Set(reflect.ValueOf(&envVarVal))
		}
	}

	return tfmr.ReverseTranslate(val)
}

// envVarName checks the dials_env tag and will use that value for the
// name. Otherwise, it will look for the dials tag and use that value with the
// optional prefix
func envVarName(prefix string, field reflect.StructField) string {
	if envTagVal := field.Tag.Get(envTagName); envTagVal != "" {
		return envTagVal
	}

	dialsTagVal := field.Tag.Get(transform.DialsTagName)
	if dialsTagVal == "" {
		panic(fmt.Errorf("Empty dials tag for field name %s", field.Name))
	}
	if prefix != "" {
		return prefix + "_" + dialsTagVal
	}

	return dialsTagVal
}
