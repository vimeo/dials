package yaml

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/tagformat"
	"github.com/vimeo/dials/transform"

	"gopkg.in/yaml.v2"
)

// YAMLTagName is the name of the `"yaml"` tag
const YAMLTagName = "yaml"

// Decoder is a decoder that knows how to work with text encoded in YAML.
type Decoder struct {
}

// Decode reads from `r` and decodes what is read as YAML depositing the
// relevant values into `t`.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	yamlBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("error reading YAML: %s", err)
	}

	tfmr := transform.NewTransformer(t.Type(),
		&tagformat.TagCopyingMangler{
			SrcTag: tagformat.DialsTagName, NewTag: YAMLTagName},
	)
	val, tfmErr := tfmr.Translate()
	if tfmErr != nil {
		return reflect.Value{}, fmt.Errorf("failed to convert tags: %s", tfmErr)
	}

	instance := val.Addr().Interface()
	err = yaml.Unmarshal(yamlBytes, instance)
	if err != nil {
		return reflect.Value{}, err
	}

	unmangledVal, unmangleErr := tfmr.ReverseTranslate(val)
	if unmangleErr != nil {
		return reflect.Value{}, unmangleErr
	}

	return unmangledVal, nil
}
