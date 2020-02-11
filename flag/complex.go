package flag

import (
	"fmt"

	"github.com/vimeo/dials/parsestring"
)

type complex128Var struct {
	c complex128
}

func (v *complex128Var) Set(s string) error {
	cmplx, err := parsestring.ParseComplex128(s)
	if err != nil {
		return err

	}
	v.c = cmplx
	return nil
}

func (v *complex128Var) String() string {
	return fmt.Sprintf("%g", v.c)
}

func (v *complex128Var) Get() interface{} {
	return v.c
}

type complex64Var struct {
	c complex64
}

func (v *complex64Var) Set(s string) error {
	cmplx, err := parsestring.ParseComplex64(s)
	if err != nil {
		return err

	}
	v.c = cmplx
	return nil
}

func (v *complex64Var) String() string {
	return fmt.Sprintf("%g", v.c)
}

func (v *complex64Var) Get() interface{} {
	return v.c
}
