package json

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/transform"
)

// JSONTagName is the name of the `"json"` tag.
const JSONTagName = "json"

// Decoder is a decoder that know how to work with text encoded in JSON
type Decoder struct {
}

// Decode is a decoder that decodes the JSON from an io.Reader into the
// appropriate struct.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	jsonBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("error reading JSON: %s", err)
	}

	// If there aren't any json tags, copy over from any dials tags.
	tfmr := transform.NewTransformer(t.Type(),
		&tagformat.TagCopyingMangler{
			SrcTag: tagformat.DialsTagName, NewTag: JSONTagName})
	val, tfmErr := tfmr.Translate()
	if tfmErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to convert tags: %s", tfmErr)
	}
	// Get a pointer to our value, so we can pass that.
	instance := val.Addr().Interface()
	err = json.Unmarshal(jsonBytes, instance)
	if err != nil {
		return reflect.Value{}, err
	}

	unmangledVal, unmangleErr := tfmr.ReverseTranslate(val)
	if unmangleErr != nil {
		return reflect.Value{}, unmangleErr
	}

	return unmangledVal, nil
}
