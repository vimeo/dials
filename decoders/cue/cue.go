package cue

import (
	"fmt"
	"io"
	"reflect"

	"cuelang.org/go/cue/cuecontext"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/transform"
)

// Decoder is a decoder that knows how to work with configs written in Cue
type Decoder struct{}

// Decode is a decoder that decodes the Cue config from an io.Reader into the
// appropriate struct.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	raw, readErr := io.ReadAll(r)
	if readErr != nil {
		return reflect.Value{}, fmt.Errorf("error reading raw bytes: %w", readErr)
	}

	const jsonTagName = "json"

	// If there aren't any json tags, copy over from any dials tags.
	tfmr := transform.NewTransformer(t.Type(),
		&tagformat.TagCopyingMangler{
			SrcTag: common.DialsTagName, NewTag: jsonTagName})
	reflVal, tfmErr := tfmr.Translate()
	if tfmErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to convert tags: %s", tfmErr)
	}

	cctxt := cuecontext.New()
	val := cctxt.CompileBytes(raw)
	if compileErr := val.Err(); compileErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to compile cue blob: %w", compileErr)
	}
	if decErr := val.Decode(reflVal.Addr().Interface()); decErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to decode cue value into dials struct: %w", decErr)
	}
	return reflVal, nil
}
