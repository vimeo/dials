// +build !go1.15

package parse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComplex128(t *testing.T) {
	for _, itbl := range []struct {
		name        string
		in          string
		expected    complex128
		expectedStr string
	}{
		{
			name:        "real_only",
			in:          "42.125",
			expected:    complex(float64(42.125), float64(0)),
			expectedStr: "(42.125+0i)",
		},
		{
			name:        "imaginary_only",
			in:          "42.125i",
			expected:    complex(float64(0), float64(42.125)),
			expectedStr: "(0+42.125i)",
		},
		{
			name:        "both_parts",
			in:          "42.125+32.5i",
			expected:    complex(float64(42.125), float64(32.5)),
			expectedStr: "(42.125+32.5i)",
		},
		{
			name:        "both_parts_without_coefficient",
			in:          "42.125+i",
			expected:    complex(float64(42.125), float64(1)),
			expectedStr: "(42.125+1i)",
		},
		{
			name:        "both_parts_negative",
			in:          "-42.125+-32.5i",
			expected:    complex(float64(-42.125), float64(-32.5)),
			expectedStr: "(-42.125-32.5i)",
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			t.Parallel()

			num, err := Complex128(tbl.in)
			require.NoError(t, err)
			assert.EqualValues(t, tbl.expected, num)
			assert.EqualValues(t, tbl.expectedStr, fmt.Sprintf("%g", num))
		})
	}

}

func TestComplex64(t *testing.T) {
	for _, itbl := range []struct {
		name        string
		in          string
		expected    complex64
		expectedStr string
	}{
		{
			name:        "real_only",
			in:          "42.125",
			expected:    complex(float32(42.125), float32(0)),
			expectedStr: "(42.125+0i)",
		},
		{
			name:        "imaginary_only",
			in:          "42.125i",
			expected:    complex(float32(0), float32(42.125)),
			expectedStr: "(0+42.125i)",
		},
		{
			name:        "both_parts",
			in:          "42.125+32.5i",
			expected:    complex(float32(42.125), float32(32.5)),
			expectedStr: "(42.125+32.5i)",
		},
		{
			name:        "both_parts_without_coefficient",
			in:          "42.125+i",
			expected:    complex(float32(42.125), float32(1)),
			expectedStr: "(42.125+1i)",
		},
		{
			name:        "both_parts_negative",
			in:          "-42.125+-32.5i",
			expected:    complex(float32(-42.125), float32(-32.5)),
			expectedStr: "(-42.125-32.5i)",
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			t.Parallel()
			num, err := Complex64(tbl.in)
			require.NoError(t, err)
			assert.EqualValues(t, tbl.expected, num)
			assert.EqualValues(t, tbl.expectedStr, fmt.Sprintf("%g", num))
		})
	}

}
