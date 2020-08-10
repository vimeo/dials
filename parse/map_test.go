package parse

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMapForStringStringMaps(t *testing.T) {
	for _, itbl := range []struct {
		name        string
		input       string
		expected    map[string]string
		expectedStr string
		expectedErr error
	}{
		{
			name:  "simple_origin",
			input: `"Origin": "foobar"`,
			expected: map[string]string{
				"Origin": "foobar",
			},
			expectedStr: `"Origin":"foobar"`,
			expectedErr: nil,
		},
		{
			name:  "origin_with_escape",
			input: `"Origin": "foobar\"fimbar"`,
			expected: map[string]string{
				"Origin": "foobar\"fimbar",
			},
			expectedStr: `"Origin":"foobar\"fimbar"`,
			expectedErr: nil,
		},
		{
			name:  "rawstring_origin",
			input: "`Origin`: `foobar`",
			expected: map[string]string{
				"Origin": "foobar",
			},
			expectedStr: `"Origin":"foobar"`,
			expectedErr: nil,
		},
		{
			name:        "origin_two_error",
			input:       `"Origin": "foobar", "Origin": "foobat"`,
			expected:    nil,
			expectedStr: ``,
			expectedErr: fmt.Errorf("map parsing failed on key %q: duplicate key %[1]q, already has value %q", "Origin", "foobar"),
		},
		{
			name:        "origin_two_referer_error",
			input:       `"Origin": "foobar", "Origin": "foobat", "Referer": "fimbat"`,
			expected:    nil,
			expectedStr: ``,
			expectedErr: fmt.Errorf("map parsing failed on key %q: duplicate key %[1]q, already has value %q", "Origin", "foobar"),
		},
		{
			name:  "origin_referer",
			input: `"Origin": "foobar", "Referer": "fimbat"`,
			expected: map[string]string{
				"Origin":  "foobar",
				"Referer": "fimbat",
			},
			expectedStr: `"Origin":"foobar","Referer":"fimbat"`,
			expectedErr: nil,
		},
		{
			name:  "origin_referer_unquoted",
			input: `Origin: foobar, Referer: fimbat`,
			expected: map[string]string{
				"Origin":  "foobar",
				"Referer": "fimbat",
			},
			expectedStr: `"Origin":"foobar","Referer":"fimbat"`,
			expectedErr: nil,
		},
		{
			name:  "paths_unquoted",
			input: `src: /etc/foo/limits.conf, src2: /etc/foo/limits3.conf`,
			expected: map[string]string{
				"src":  "/etc/foo/limits.conf",
				"src2": "/etc/foo/limits3.conf",
			},
			expectedStr: `"src":"/etc/foo/limits.conf","src2":"/etc/foo/limits3.conf"`,
			expectedErr: nil,
		},
		{
			name:  "prices_unquoted",
			input: `us:$22.33,uk:33.44£,ja:777¥`,
			expected: map[string]string{
				"us": "$22.33", "uk": "33.44£", "ja": "777¥",
			},
			expectedStr: `us:$22.33,uk:33.44£,ja:777¥`,
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
			expected:    map[string]string{},
			expectedStr: "",
			expectedErr: nil,
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			ss, err := Map(tbl.input, reflect.TypeOf(map[string]string{}))
			if tbl.expectedErr != nil {
				assert.EqualError(t, err, tbl.expectedErr.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, ss)
			assert.EqualValues(t, tbl.expected, ss.Interface().(map[string]string))
		})
	}
}
