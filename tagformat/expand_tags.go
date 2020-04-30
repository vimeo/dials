package tagformat

import (
	"reflect"
	"strconv"

	"github.com/vimeo/dials/transform"
)

// TagCopyingMangler implements the transform.Mangler interface
// copying `SrcTag` tags unmodified into a new tag specified by
// the `NewTag` member, for example `json` or `yaml`. It is intended to be
// used by `dials.Decoder`s which need to make use of those tags. To change the
// casing of tags, use a dialsTagReformattingSource.
type TagCopyingMangler struct {
	SrcTag, NewTag string
}

// Mangle is called for every field in a struct, and maps that to one or more output fields.
// Implementations that desire to leave fields unchanged should return
// the argument unchanged. (particularly useful if taking advantage of
// recursive evaluation)
func (t *TagCopyingMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	srcVal := sf.Tag.Get(t.SrcTag)
	if srcVal == "" {
		return []reflect.StructField{sf}, nil
	}
	currentNewTagVal := sf.Tag.Get(t.NewTag)
	if currentNewTagVal != "" {
		return []reflect.StructField{sf}, nil
	}

	newTags := string(sf.Tag)

	if len(newTags) > 0 {
		newTags += " "
	}

	newTags += t.NewTag + ":" + strconv.Quote(srcVal)

	sf.Tag = reflect.StructTag(newTags)

	return []reflect.StructField{sf}, nil
}

// Unmangle is called for every source-field->mangled-field
// mapping-set, with the mangled-field and its populated value set.
// This just returns the first field, as Mangle only returns one field at a
// time.
func (t *TagCopyingMangler) Unmangle(sf reflect.StructField, vs []transform.FieldValueTuple) (reflect.Value, error) {
	return vs[0].Value, nil
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
// This ShouldRecurse always returns true here so inner struct tags get
// appropriately mangled.
func (t *TagCopyingMangler) ShouldRecurse(_ reflect.StructField) bool {
	return true
}
