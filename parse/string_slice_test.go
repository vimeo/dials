package parse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStringSlice(t *testing.T) {
	for _, itbl := range []struct {
		name        string
		input       string
		expected    []string
		expectedStr string
		expectedErr error
	}{
		{
			name:        "empty",
			input:       "",
			expected:    []string{},
			expectedStr: "",
			expectedErr: nil,
		},
		{
			name:        "one_ident",
			input:       "a",
			expected:    []string{"a"},
			expectedStr: "\"a\"",
			expectedErr: nil,
		},
		{
			name:        "two_idents",
			input:       "a,b",
			expected:    []string{"a", "b"},
			expectedStr: "\"a\",\"b\"",
			expectedErr: nil,
		},
		{
			name:        "one_ident_one_int",
			input:       "a,33",
			expected:    []string{"a", "33"},
			expectedStr: "\"a\",\"33\"",
			expectedErr: nil,
		},
		{
			name:        "one_ident_one_float",
			input:       "a,33.0",
			expected:    []string{"a", "33.0"},
			expectedStr: "\"a\",\"33.0\"",
			expectedErr: nil,
		},
		{
			name:        "two_strings",
			input:       `"a","b"`,
			expected:    []string{"a", "b"},
			expectedStr: `"a","b"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_comma",
			input:       `"a","b,"`,
			expected:    []string{"a", "b,"},
			expectedStr: `"a","b,"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas",
			input:       `",a","b,"`,
			expected:    []string{",a", "b,"},
			expectedStr: `",a","b,"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas_and_escaped_quotes",
			input:       "\",a\",\"b,\\\"\"",
			expected:    []string{",a", "b,\""},
			expectedStr: `",a","b,\""`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas_and_raw_quotes",
			input:       "`a,`, `,b`",
			expected:    []string{"a,", ",b"},
			expectedStr: `"a,",",b"`,
			expectedErr: nil,
		},
		{
			name:        "unclosed_quotes",
			input:       "`a,`, `,b",
			expected:    nil,
			expectedStr: "",
			expectedErr: fmt.Errorf(
				"parsing failed: map[<input>:1:10:literal not terminated]"),
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			out, err := ParseStringSlice(tbl.input)

			if tbl.expectedErr != nil {
				assert.EqualError(t, err, tbl.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, out)
			assert.EqualValues(t, tbl.expected, out)
		})
	}
}
