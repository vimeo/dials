package static

import (
	"context"
	"reflect"
	"strings"

	"github.com/vimeo/dials"
)

// StringSource gets data from a string
type StringSource struct {
	Data    string
	Decoder dials.Decoder
}

var _ dials.Source = (*StringSource)(nil)

// Value ...
func (s *StringSource) Value(_ context.Context, t *dials.Type) (reflect.Value, error) {
	reader := strings.NewReader(s.Data)
	return s.Decoder.Decode(reader, t)
}
