package transform

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/fatih/structtag"
	"github.com/vimeo/dials/common"
)

const (
	dialsAliasTagSuffix = "alias"
	aliasFieldSuffix    = "_alias9wr876rw3" // a random string to append to the alias field to avoid collisions
)

// AliasMangler manages aliases for dials, dialsenv, dialsflag, and dialspflag
// struct tags to make it possible to migrate from one name to another
// conveniently.
type AliasMangler struct {
	tags []string
}

// NewAliasMangler creates a new AliasMangler with the provided tags.
func NewAliasMangler(tags ...string) *AliasMangler {
	return &AliasMangler{tags: tags}
}

// Mangle implements the Mangler interface.  If any alias tag is defined, the
// struct field will be copied with the non-aliased tag set to the alias's
// value.
func (a *AliasMangler) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	originalVals := map[string]string{}
	aliasVals := map[string]string{}

	sfTags, parseErr := structtag.Parse(string(sf.Tag))
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing source tags %w", parseErr)
	}

	anyAliasFound := false
	for _, tag := range a.tags {
		if originalVal, getErr := sfTags.Get(tag); getErr == nil {
			originalVals[tag] = originalVal.Name
		}

		if aliasVal, getErr := sfTags.Get(tag + dialsAliasTagSuffix); getErr == nil {
			aliasVals[tag] = aliasVal.Name
			anyAliasFound = true

			// remove the alias tag from the definition
			sfTags.Delete(tag + dialsAliasTagSuffix)
		}
	}

	if !anyAliasFound {
		// we didn't find any aliases so just get out early
		return []reflect.StructField{sf}, nil
	}

	aliasField := sf
	aliasField.Name += aliasFieldSuffix

	// now that we've copied it, reset the struct tags on the source field to
	// not include the alias tags
	sf.Tag = reflect.StructTag(sfTags.String())

	tags, parseErr := structtag.Parse(string(aliasField.Tag))
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing struct tags: %w", parseErr)
	}

	// keep track of the aliases we actually set so we can update the dialsdesc
	setAliases := []string{}

	for tag, aliasVal := range aliasVals {
		// remove the alias tag so it's not left on the copied StructField
		tags.Delete(tag + dialsAliasTagSuffix)

		newDialsTag := &structtag.Tag{
			Key:  tag,
			Name: aliasVal,
		}

		if setErr := tags.Set(newDialsTag); setErr != nil {
			return nil, fmt.Errorf("error setting new value for dials tag: %w", setErr)
		}
		setAliases = append(setAliases, tag+"="+originalVals[tag])
	}

	newDialsDesc := "base dialsdesc unset" // be pessimistic in case dialsdesc isn't set
	if desc, getErr := tags.Get(common.DialsHelpTextTag); getErr == nil {
		newDialsDesc = desc.Name
	}

	sort.Strings(setAliases)
	newDesc := &structtag.Tag{
		Key:  common.DialsHelpTextTag,
		Name: newDialsDesc + " (alias of " + strings.Join(setAliases, " ") + ")",
	}
	if setErr := tags.Set(newDesc); setErr != nil {
		return nil, fmt.Errorf("error setting amended dialsdesc for tag %w", setErr)
	}

	// set the new tags on the alias field
	aliasField.Tag = reflect.StructTag(tags.String())

	return []reflect.StructField{sf, aliasField}, nil
}

// Unmangle implements the Mangler interface and unwinds the alias copying
// operation.  Note that if both the source and alias are both set in the
// configuration, an error will be returned.
func (a *AliasMangler) Unmangle(sf reflect.StructField, fvs []FieldValueTuple) (reflect.Value, error) {
	switch len(fvs) {
	case 1:
		// if there's only one tuple that means there was no alias, so just
		// return...
		return fvs[0].Value, nil
	case 2:
		// two means there's an alias so we should continue on...
	default:
		return reflect.Value{}, fmt.Errorf("expected 1 or 2 tuples, got %d", len(fvs))
	}

	if !fvs[0].Value.IsNil() && !fvs[1].Value.IsNil() {
		return reflect.Value{}, fmt.Errorf("both alias and original set for field %q", sf.Name)
	}

	// return the first one that isn't nil
	for _, fv := range fvs {
		if !fv.Value.IsNil() {
			return fv.Value, nil
		}
	}

	// if we made it this far, they were both nil, which is fine -- just return
	// one of them.
	return fvs[0].Value, nil
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
func (a AliasMangler) ShouldRecurse(_ reflect.StructField) bool {
	return true
}
