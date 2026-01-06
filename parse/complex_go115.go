//go:build go1.15

package parse

import (
	"strconv"
)

// Complex64 delegates to strconv.ParseComplex for go >= 1.15.
func Complex64(s string) (complex64, error) {
	c, err := strconv.ParseComplex(s, 64)
	return complex64(c), err
}

// Complex128 delegates to strconv.ParseComplex for go >= 1.15.
func Complex128(s string) (complex128, error) {
	return strconv.ParseComplex(s, 128)
}
