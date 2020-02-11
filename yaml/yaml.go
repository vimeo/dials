package yaml

import (
	"io"
	"reflect"

	"github.com/vimeo/dials"
	"gopkg.in/yaml.v2"
)

// Decoder is a decoder that knows how to work with text encoded in YAML.
type Decoder struct {
}

// Decode reads from `r` and decodes what is read as YAML depositing the
// relevant values into `t`.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	decoder := yaml.NewDecoder(r)

	instance := reflect.New(t.Type()).Interface()
	err := decoder.Decode(instance)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf(instance), nil
}
