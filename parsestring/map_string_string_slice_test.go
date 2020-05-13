package parsestring

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseMapSlice(t *testing.T) {
	for _, itbl := range []struct {
		name        string
		input       string
		expected    map[string][]string
		expectedErr error
	}{
		{
			name:  "simple_origin",
			input: `"Origin": "foobar"`,
			expected: map[string][]string{
				"Origin": {"foobar"},
			},
			expectedErr: nil,
		},
		{
			name:  "origin_with_escape",
			input: `"Origin": "foobar\"fimbar"`,
			expected: map[string][]string{
				"Origin": {"foobar\"fimbar"},
			},
			expectedErr: nil,
		},
		{
			name:  "rawstring_origin",
			input: "`Origin`: `foobar`",
			expected: map[string][]string{
				"Origin": {"foobar"},
			},
			expectedErr: nil,
		},
		{
			name:  "origin_two",
			input: `"Origin": "foobar", "Origin": "foobat"`,
			expected: map[string][]string{
				"Origin": {"foobar", "foobat"},
			},
			expectedErr: nil,
		},
		{
			name:  "origin_two_referer",
			input: `"Origin": "foobar", "Origin": "foobat", "Referer": "fimbat"`,
			expected: map[string][]string{
				"Origin":  {"foobar", "foobat"},
				"Referer": {"fimbat"},
			},
			expectedErr: nil,
		},
		{
			name:        "naked_colon",
			input:       `:`,
			expected:    nil,
			expectedErr: fmt.Errorf("unexpected colon"),
		},
		{
			name:        "naked_comma",
			input:       `,`,
			expected:    map[string][]string{},
			expectedErr: nil,
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			out, err := ParseStringStringSliceMap(tbl.input)

			if tbl.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error (%s), got nil err and value %v",
						tbl.expectedErr, out)
				} else if err.Error() != tbl.expectedErr.Error() {
					t.Errorf("unexpected error: %s; expected: %s",
						err, tbl.expectedErr)
				}
				return
			}

			if !reflect.DeepEqual(tbl.expected, out) {
				t.Errorf("unexpected output: %v; expected %v", out, tbl.expected)
			}

		})
	}
}
