package tagformat

import (
	"reflect"

	"github.com/fatih/structtag"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/transform"
)

// TagReformattingMangler implements transform.Mangler, rewriting the specified
// struct tag (with Mangler recursion enabled)
type TagReformattingMangler struct {
	tag              string
	decodeCasingFunc caseconversion.DecodeCasingFunc
	encodeCasingFunc caseconversion.EncodeCasingFunc
}

// NewTagReformattingMangler constructs a new TagReformattingMangler that can
// replace the value of an existing tag
func NewTagReformattingMangler(tagName string,
	enc caseconversion.DecodeCasingFunc, dec caseconversion.EncodeCasingFunc) *TagReformattingMangler {
	return &TagReformattingMangler{
		tag:              tagName,
		decodeCasingFunc: enc,
		encodeCasingFunc: dec,
	}
}

// Mangle is called for every field in a struct, and returns the value
// unchanged other than replacing the specified tag.
func (k *TagReformattingMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	nameVal := sf.Tag.Get(k.tag)
	dcf := k.decodeCasingFunc
	if nameVal == "" {
		// There was no name defined, so just fall back to the field name and
		// force the decoder to use Go conventions.
		nameVal = sf.Name
		dcf = caseconversion.DecodeGoCamelCase
	}

	splitTagVal, err := dcf(nameVal)
	if err != nil {
		return nil, err
	}

	encodedTagVal := k.encodeCasingFunc(splitTagVal)

	tags, parseErr := structtag.Parse(string(sf.Tag))
	if parseErr != nil {
		return nil, err
	}
	tags.Set(&structtag.Tag{
		Key:     k.tag,
		Name:    encodedTagVal,
		Options: []string{},
	})

	sf.Tag = reflect.StructTag(tags.String())
	return []reflect.StructField{sf}, nil
}

// Unmangle is called for every source-field->mangled-field
// mapping-set, with the mangled-field and its populated value set. The
// implementation of Unmangle should return a reflect.Value that will
// be used for the next mangler or final struct value)
func (k *TagReformattingMangler) Unmangle(_ reflect.StructField, vs []transform.FieldValueTuple) (reflect.Value, error) {
	// we always return exactly one field in Mangle, so we can always return vs[0].Value without any issues
	return vs[0].Value, nil
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
func (k *TagReformattingMangler) ShouldRecurse(_ reflect.StructField) bool {
	return true
}

// ReformatDialsTagSource is a convenience function that provides a source
// that first reformats `dials` tags on the passed type for a dials
// config struct, changing the casing as specified (e.g., from lowerCamelCase
// into snake_case), then calls Value on the wrapped source with the modified
// struct type passed in.
func ReformatDialsTagSource(inner dials.Source, encodeFunc caseconversion.DecodeCasingFunc, decodeFunc caseconversion.EncodeCasingFunc) dials.Source {
	return sourcewrap.NewTransformingSource(inner, &TagReformattingMangler{
		tag:              common.DialsTagName,
		decodeCasingFunc: encodeFunc,
		encodeCasingFunc: decodeFunc,
	})

}
