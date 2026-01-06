//go:build !go1.15

package parse

import (
	"fmt"
	"strconv"
	"strings"
)

// Complex64 turns a string representation of a complex number into a complex64.
func Complex64(s string) (complex64, error) {
	if plusCount := strings.Count(s, "+"); plusCount > 1 {
		return 0, fmt.Errorf("too many '+'s: %d, at most 1 allowed",
			plusCount)
	}
	var (
		rpart, ipart float32
	)
	parts := strings.SplitN(s, "+", 2)
	for d, part := range parts {
		stripped := strings.TrimSpace(part)
		pv, isimag, err := parsePart64(stripped)
		if err != nil {
			return 0, fmt.Errorf("failed to parse %d part: %s", d, err)
		}
		switch isimag {
		case true:
			ipart = pv
		case false:
			rpart = pv
		}
	}

	return complex(rpart, ipart), nil
}

// bool indicates whether the result is from the imaginary part
func parsePart64(part string) (float32, bool, error) {
	var isImag bool
	if len(part) == 0 {
		return 0, isImag, fmt.Errorf("empty value for complex component")
	}
	if part[len(part)-1] == 'i' {
		isImag = true
		// We just need the coefficient of the imaginary part
		part = strings.TrimSpace(part[:len(part)-1])
		if len(part) == 0 {
			return 1, true, nil
		}
	}
	val, parseErr := strconv.ParseFloat(part, 32)
	if parseErr != nil {
		componentName := "real"
		if isImag {
			componentName = "imaginary"
		}
		return 0, isImag, fmt.Errorf("failed to parse %s part %q as float: %s",
			componentName, part, parseErr)
	}
	return float32(val), isImag, nil
}

// Complex128 turns a string representation of a complex number into a complex128.
func Complex128(s string) (complex128, error) {
	if plusCount := strings.Count(s, "+"); plusCount > 1 {
		return 0, fmt.Errorf("too many '+'s: %d, at most 1 allowed",
			plusCount)
	}
	var (
		rpart, ipart float64
	)
	parts := strings.SplitN(s, "+", 2)
	for d, part := range parts {
		stripped := strings.TrimSpace(part)
		pv, isimag, err := parsePart128(stripped)
		if err != nil {
			return 0, fmt.Errorf("failed to parse %d part: %s", d, err)
		}
		switch isimag {
		case true:
			ipart = pv
		case false:
			rpart = pv
		}
	}

	return complex(rpart, ipart), nil
}

// bool indicates whether the result is from the imaginary part
func parsePart128(part string) (float64, bool, error) {
	if len(part) == 0 {
		return 0, false, fmt.Errorf("empty value for complex component")
	}
	isImag := false
	if part[len(part)-1] == 'i' {
		isImag = true
		// We just need the coefficient of the imaginary part
		part = strings.TrimSpace(part[:len(part)-1])
		if len(part) == 0 {
			return 1, true, nil
		}
	}
	val, parseErr := strconv.ParseFloat(part, 64)
	if parseErr != nil {
		componentName := "real"
		if isImag {
			componentName = "imaginary"
		}
		return 0, isImag, fmt.Errorf("failed to parse %s part %q as float: %s",
			componentName, part, parseErr)
	}
	return val, isImag, nil
}
