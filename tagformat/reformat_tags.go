package tagformat

import (
	"reflect"

	"github.com/fatih/structtag"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/sourcewrap"
	"github.com/vimeo/dials/transform"
)

// DecodeCasingFunc takes in an identifier in a case such as camelCase or
// snake_case and splits it up into a DecodedIdentifier for encoding by an
// EncodeCasingFunc into a different case.
type DecodeCasingFunc func(string) (DecodedIdentifier, error)

// EncodeCasingFunc combines the contents of a DecodedIdentifier into an
// identifier in a case such as camelCase or snake_case.
type EncodeCasingFunc func(DecodedIdentifier) string

// DecodedIdentifier is an slice of lowercase words (e.g., []string{"test",
// "string"}) produced by a DecodeCasingFunc, which can be encoded by an
// EncodeCasingFunc into a string in the specified case (e.g., with
// EncodeLowerCamelCase, "testString").
type DecodedIdentifier []string

// TagReformattingMangler implements transform.Mangler, rewriting the specified
// struct tag (with Mangler recursion enabled)
type TagReformattingMangler struct {
	tag              string
	decodeCasingFunc DecodeCasingFunc
	encodeCasingFunc EncodeCasingFunc
}

// NewTagReformattingMangler constructs a new TagReformattingMangler that can
// replace the value of an existing tag
func NewTagReformattingMangler(tagName string,
	enc DecodeCasingFunc, dec EncodeCasingFunc) *TagReformattingMangler {
	return &TagReformattingMangler{
		tag:              tagName,
		decodeCasingFunc: enc,
		encodeCasingFunc: dec,
	}
}

// Mangle is called for every field in a struct, and returns the value
// unchanged other than replacing the specified tag.
func (k *TagReformattingMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	dialsVal := sf.Tag.Get(k.tag)
	if dialsVal == "" {
		return []reflect.StructField{sf}, nil
	}
	splitTagVal, err := k.decodeCasingFunc(dialsVal)
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

// ReformatdialsTagSource is a convenience function that provides a source
// that first reformats `dials` tags on the passed type for a dials
// config struct, changing the casing as specified (e.g., from lowerCamelCase
// into snake_case), then calls Value on the wrapped source with the modified
// struct type passed in.
func ReformatdialsTagSource(inner dials.Source, encodeFunc DecodeCasingFunc, decodeFunc EncodeCasingFunc) dials.Source {
	return sourcewrap.NewTransformingSource(inner, &TagReformattingMangler{
		tag:              DialsTagName,
		decodeCasingFunc: encodeFunc,
		encodeCasingFunc: decodeFunc,
	})

}
