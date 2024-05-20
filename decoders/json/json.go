package json

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/decoders/json/jsontypes"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/transform"
)

// JSONTagName is the name of the `"json"` tag.
const JSONTagName = "json"

// Decoder is a decoder that knows how to work with text encoded in JSON
type Decoder struct{}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// pre-declare the time.Duration -> jsontypes.ParsingDuration mangler at
// package-scope, so we don't have to construct a new one every time Decode is
// called.
var parsingDurMangler = must(transform.NewSingleTypeSubstitutionMangler[time.Duration, jsontypes.ParsingDuration]())

// Decode is a decoder that decodes the JSON from an io.Reader into the
// appropriate struct.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	jsonBytes, err := io.ReadAll(r)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("error reading JSON: %s", err)
	}

	// If there aren't any json tags, copy over from any dials tags.
	tfmr := transform.NewTransformer(t.Type(),
		parsingDurMangler,
		&tagformat.TagCopyingMangler{
			SrcTag: common.DialsTagName, NewTag: JSONTagName})
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
