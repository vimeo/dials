package toml

import (
	"io"
	"reflect"

	tomlparser "github.com/pelletier/go-toml"
	"github.com/vimeo/dials"
)

// Decoder is a decoder than understands TOML.
type Decoder struct {
}

// Decode will read from `r` and parse it as TOML depositing the relevant values
// in `t`.
func (d *Decoder) Decode(r io.Reader, t *dials.Type) (reflect.Value, error) {
	decoder := tomlparser.NewDecoder(r)

	instance := reflect.New(t.Type()).Interface()
	err := decoder.Decode(instance)
	if err != nil {
		return reflect.Value{}, err
	}

	return reflect.ValueOf(instance), nil
}
