package flaghelper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vimeo/dials/parse"
)

// SignedInt represents all signed integer types
type SignedInt interface {
	int8 | int16 | int32 | int64 | int
}

// SignedIntegralSliceFlag is a wrapper around an integral-typed slice
type SignedIntegralSliceFlag[I SignedInt] struct {
	s         *[]I
	defaulted bool
}

// NewSignedIntegralSlice is a constructor for NewSignedIntegralSliceFlag
func NewSignedIntegralSlice[I SignedInt](s *[]I) *SignedIntegralSliceFlag[I] {
	return &SignedIntegralSliceFlag[I]{s: s, defaulted: true}
}

// Set implements pflag.Value and flag.Value
func (v *SignedIntegralSliceFlag[I]) Set(s string) error {
	parsed, err := parse.SignedIntegralSlice[I](s)
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
func (v *SignedIntegralSliceFlag[I]) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *SignedIntegralSliceFlag[I]) String() string {
	if v.s == nil {
		return ""
	}
	b := strings.Builder{}
	for i, z := range *v.s {
		b.WriteString(strconv.FormatInt(int64(z), 10))
		if i < len(*v.s)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

// Type implements pflag.Value
func (v *SignedIntegralSliceFlag[I]) Type() string {
	return fmt.Sprintf("%T", v.s)
}
