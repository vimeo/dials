package parse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStringSet(t *testing.T) {
	sv := func(vs ...string) map[string]struct{} {
		out := make(map[string]struct{}, len(vs))
		for _, v := range vs {
			out[v] = struct{}{}
		}
		return out
	}
	for _, itbl := range []struct {
		name        string
		input       string
		expected    map[string]struct{}
		expectedStr string
		expectedErr error
	}{
		{
			name:        "empty",
			input:       "",
			expected:    sv(),
			expectedStr: "",
			expectedErr: nil,
		},
		{
			name:        "one_ident",
			input:       "a",
			expected:    sv("a"),
			expectedStr: "\"a\"",
			expectedErr: nil,
		},
		{
			name:        "two_idents",
			input:       "a,b",
			expected:    sv("a", "b"),
			expectedStr: "\"a\",\"b\"",
			expectedErr: nil,
		},
		{
			name:        "one_ident_one_int",
			input:       "a,33",
			expected:    sv("a", "33"),
			expectedStr: "\"33\",\"a\"",
			expectedErr: nil,
		},
		{
			name:        "one_ident_one_float",
			input:       "a,33.0",
			expected:    sv("a", "33.0"),
			expectedStr: `"33.0","a"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings",
			input:       `"a","b"`,
			expected:    sv("a", "b"),
			expectedStr: `"a","b"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_comma",
			input:       `"a","b,"`,
			expected:    sv("a", "b,"),
			expectedStr: `"a","b,"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas",
			input:       `",a","b,"`,
			expected:    sv(",a", "b,"),
			expectedStr: `",a","b,"`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas_and_escaped_quotes",
			input:       "\",a\",\"b,\\\"\"",
			expected:    sv(",a", "b,\""),
			expectedStr: `",a","b,\""`,
			expectedErr: nil,
		},
		{
			name:        "two_strings_with_commas_and_raw_quotes",
			input:       "`a,`, `,b`",
			expected:    sv("a,", ",b"),
			expectedStr: `",b","a,"`,
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
		{
			name:        "duplicate_value",
			input:       "a,a,a",
			expected:    nil,
			expectedStr: "",
			expectedErr: fmt.Errorf(
				"failed to add val %q: %[1]q already present in set", "a"),
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			out, err := ParseStringSet(tbl.input)
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
