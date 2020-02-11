package static

import (
	"reflect"
	"strings"

	"github.com/vimeo/dials"
)

// StringSource gets data from a string
type StringSource struct {
	Data    string
	Decoder dials.Decoder
}

// Value ...
func (s *StringSource) Value(t *dials.Type) (reflect.Value, error) {
	reader := strings.NewReader(s.Data)
	return s.Decoder.Decode(reader, t)
}
