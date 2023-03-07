package flaghelper

import (
	"fmt"

	"github.com/vimeo/dials/parse"
)

// Complex128Var is a complex128 wrapper
type Complex128Var struct {
	c *complex128
}

// NewComplex128Var is the constructor for Complex128Var
func NewComplex128Var(c *complex128) *Complex128Var {
	return &Complex128Var{c: c}
}

// Set implement pflag.Value and flag.Value
func (v *Complex128Var) Set(s string) error {
	cmplx, err := parse.Complex128(s)
	if err != nil {
		return err

	}
	*v.c = cmplx
	return nil
}

// Get implements flag.Value
func (v *Complex128Var) Get() interface{} {
	return v.c
}

// String implements flag.Value and pflag.Value
func (v *Complex128Var) String() string {
	if v.c == nil {
		return ""
	}
	return fmt.Sprintf("%g", *v.c)
}

// Type implements pflag.Value
func (v *Complex128Var) Type() string {
	return fmt.Sprintf("%T", *v.c)
}

// Complex64Var is a wrapper around complex64
type Complex64Var struct {
	c *complex64
}

// NewComplex64Var is the constructor for Complex64Var
func NewComplex64Var(c *complex64) *Complex64Var {
	return &Complex64Var{c: c}
}

// Set implement pflag.Value and flag.Value
func (v *Complex64Var) Set(s string) error {
	cmplx, err := parse.Complex64(s)
	if err != nil {
		return err

	}
	*v.c = cmplx
	return nil
}

// Get implements flag.Value
func (v *Complex64Var) Get() interface{} {
	return v.c
}

// String implements flag.Value and pflag.Value
func (v *Complex64Var) String() string {
	if v.c == nil {
		return ""
	}
	return fmt.Sprintf("%g", *v.c)
}

// Type implements pflag.Value
func (v *Complex64Var) Type() string {
	return fmt.Sprintf("%T", *v.c)
}
