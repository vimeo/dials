package parse

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// SignedIntegralSlice splits on commas and parses into a slice of integers
// Parses with strconv.ParseInt and the base set to 0 so base prefixes are available.
// Whitespace is trimmed around the integers before parsing to allow for reasonable separtion (shell word-splitting aside)
func SignedIntegralSlice[I int | int64 | int32 | int16 | int8](s string) ([]I, error) {
	parts := strings.Split(s, ",")
	out := make([]I, len(parts))

	bitSize := int(unsafe.Sizeof(I(0)) * 8)

	for i, p := range parts {
		val, parseErr := strconv.ParseInt(strings.TrimSpace(p), 0, bitSize)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse integer index %d: %w", i, parseErr)
		}
		out[i] = I(val)
	}
	return out, nil
}

// UnsignedIntegralSlice splits on commas and parses into a slice of integers
// Parses with strconv.ParseInt and the base set to 0 so base prefixes are available.
// Whitespace is trimmed around the integers before parsing to allow for reasonable separtion (shell word-splitting aside)
func UnsignedIntegralSlice[I uint | uint64 | uint32 | uint16 | uint8 | uintptr](s string) ([]I, error) {
	parts := strings.Split(s, ",")
	out := make([]I, len(parts))

	bitSize := int(unsafe.Sizeof(I(0)) * 8)

	for i, p := range parts {
		val, parseErr := strconv.ParseUint(strings.TrimSpace(p), 0, bitSize)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse integer index %d: %w", i, parseErr)
		}
		out[i] = I(val)
	}
	return out, nil
}
