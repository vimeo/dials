package flaghelper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vimeo/dials/parse"
)

// UnsignedInt represents all unsigned integer types
type UnsignedInt interface {
	uint8 | uint16 | uint32 | uint64 | uint | uintptr
}

// UnsignedIntegralSliceFlag is a wrapper around an unsigned integral-typed slice
type UnsignedIntegralSliceFlag[I UnsignedInt] struct {
	s         *[]I
	defaulted bool
}

// NewUnsignedIntegralSlice is a constructor for StringSliceFlag
func NewUnsignedIntegralSlice[I UnsignedInt](s *[]I) *UnsignedIntegralSliceFlag[I] {
	return &UnsignedIntegralSliceFlag[I]{s: s, defaulted: true}
}

// Set implements pflag.Value and flag.Value
func (v *UnsignedIntegralSliceFlag[I]) Set(s string) error {
	parsed, err := parse.UnsignedIntegralSlice[I](s)
	if err != nil {
		return err
	}
	if v.defaulted {
		*v.s = parsed
		v.defaulted = false
		return nil
	}
	*v.s = append(*v.s, parsed...)
	return nil
}

// Get implements flag.Value
func (v *UnsignedIntegralSliceFlag[I]) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *UnsignedIntegralSliceFlag[I]) String() string {
	if v.s == nil {
		return ""
	}
	b := strings.Builder{}
	for i, z := range *v.s {
		b.WriteString(strconv.FormatUint(uint64(z), 10))
		if i < len(*v.s)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

// Type implements pflag.Value
func (v *UnsignedIntegralSliceFlag[I]) Type() string {
	return fmt.Sprintf("%T", v.s)
}
