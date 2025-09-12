package flaghelper

import (
	"encoding"
	"fmt"
)

// MarshalWrapper wraps TextUnmarshaler
type MarshalWrapper struct {
	v encoding.TextUnmarshaler
}

// NewMarshalWrapper is the constructor for MarshalWrapper
func NewMarshalWrapper(v encoding.TextUnmarshaler) *MarshalWrapper {
	return &MarshalWrapper{
		v: v,
	}
}

func (w MarshalWrapper) String() string {
	if m, ok := w.v.(encoding.TextMarshaler); ok {
		b, err := m.MarshalText()
		if err == nil {
			return string(b)
		}
	}
	if s, ok := w.v.(fmt.Stringer); ok {
		return s.String()
	}
	return ""
}

// Set implements flag.Value and pflag.Value
func (w MarshalWrapper) Set(s string) error {
	return w.v.UnmarshalText([]byte(s))
}

// Get implements flag.Value
func (w MarshalWrapper) Get() any {
	return w.v
}

// Type implements pflag.Value
func (w MarshalWrapper) Type() string {
	return fmt.Sprintf("%T", w.v)
}
