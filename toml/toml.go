package toml

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/transform"

	tomlparser "github.com/pelletier/go-toml"
)

// TOMLTagName is the name of the `"toml"` tag.
const TOMLTagName = "toml"

// Decoder is a decoder than understands TOML.
type Decoder struct {
}

// Decode will read from `r` and parse it as TOML depositing the relevant values
// in `t`.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	tomlBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("error reading TOML: %s", err)
	}

	// Use the TagCopyingMangler to copy over TOML tags from dials tags if TOML
	// tags aren't specified.
	tfmr := transform.NewTransformer(t.Type(),
		&tagformat.TagCopyingMangler{
			SrcTag: tagformat.DialsTagName, NewTag: TOMLTagName})
	val, tfmErr := tfmr.Translate()
	if tfmErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to convert tags: %s", tfmErr)
	}

	// Get a pointer to our value, so we can pass that.
	instance := val.Addr().Interface()
	err = tomlparser.Unmarshal(tomlBytes, instance)
	if err != nil {
		return reflect.Value{}, err
	}

	unmangledVal, unmangleErr := tfmr.ReverseTranslate(val)
	if unmangleErr != nil {
		return reflect.Value{}, unmangleErr
	}

	return unmangledVal, nil
}
