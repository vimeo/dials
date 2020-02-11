package flag

import (
	"encoding"
	"fmt"
)

type marshalWrapper struct {
	v encoding.TextUnmarshaler
}

func (w marshalWrapper) String() string {
	if s, ok := w.v.(fmt.Stringer); ok {
		return s.String()
	}
	return ""
}

func (w marshalWrapper) Set(s string) error {
	return w.v.UnmarshalText([]byte(s))
}

func (w marshalWrapper) Get() interface{} {
	return w.v
}
