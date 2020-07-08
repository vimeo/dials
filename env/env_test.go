package env

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
)

func TestEnv(t *testing.T) {
	type Embed struct {
		Foo int
		Bar bool
	}
	cases := map[string]struct {
		ConfigStruct interface{}
		EnvVarName   string
		EnvVarValue  string
		Source       Source
		Expected     interface{}
		ExpectedErr  string
	}{
		"string": {
			ConfigStruct: &struct{ EnvVar string }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "asdf",
			Expected:     &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_dials_tag": {
			ConfigStruct: &struct {
				EnvVar string `dials:"ENVIRONMENT_VARIABLE"`
			}{},
			EnvVarName:  "ENVIRONMENT_VARIABLE",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_env_tag": {
			ConfigStruct: &struct {
				EnvVar string `dials_env:"ENVIRONMENT_VARIABLE"`
			}{},
			EnvVarName:  "ENVIRONMENT_VARIABLE",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_dials_and_env_tags": {
			ConfigStruct: &struct {
				EnvVar string `dials:"env-var" dials_env:"ENV_TWO"`
			}{},
			EnvVarName:  "ENV_TWO",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"int": {
			ConfigStruct: &struct{ EnvVar int }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "123",
			Expected:     &struct{ EnvVar int }{EnvVar: 123},
		},
		"int8": {
			ConfigStruct: &struct{ EnvVar int8 }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "123",
			Expected:     &struct{ EnvVar int8 }{EnvVar: int8(123)},
		},
		"int8_overflow": {
			ConfigStruct: &struct{ EnvVar int8 }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "999999",
			Expected:     nil,
			ExpectedErr:  "Overflow of int8 type: 999999",
		},
		"bool": {
			ConfigStruct: &struct{ EnvVar bool }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "true",
			Expected:     &struct{ EnvVar bool }{EnvVar: true},
		},
		"float32": {
			ConfigStruct: &struct{ EnvVar float32 }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "1.123",
			Expected:     &struct{ EnvVar float32 }{EnvVar: float32(1.123)},
		},
		"string_slice": {
			ConfigStruct: &struct{ EnvVar []string }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  `"a","b"`,
			Expected:     &struct{ EnvVar []string }{EnvVar: []string{"a", "b"}},
		},
		"prefixed_string": {
			ConfigStruct: &struct{ EnvVar string }{},
			EnvVarName:   "PREFIX_ENV_VAR",
			EnvVarValue:  "asdf",
			Source:       Source{Prefix: "PREFIX"},
			Expected:     &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"zero_value_string": {
			ConfigStruct: &struct{ EnvVar string }{},
			EnvVarName:   "NOT_STRUCT_FIELD_NAME",
			EnvVarValue:  "asdf",
			Expected:     &struct{ EnvVar string }{EnvVar: ""},
		},
		"zero_value_int": {
			ConfigStruct: &struct{ EnvVar int }{},
			EnvVarName:   "NOT_STRUCT_FIELD_NAME",
			EnvVarValue:  "123",
			Expected:     &struct{ EnvVar int }{EnvVar: 0},
		},
		"zero_value_bool": {
			ConfigStruct: &struct{ EnvVar bool }{},
			EnvVarName:   "NOT_STRUCT_FIELD_NAME",
			EnvVarValue:  "true",
			Expected:     &struct{ EnvVar bool }{EnvVar: false},
		},
		"zero_value_string_slice": {
			ConfigStruct: &struct{ EnvVar []string }{},
			EnvVarName:   "ENV_VAR",
			EnvVarValue:  "",
			Expected:     &struct{ EnvVar []string }{EnvVar: []string{}},
		},
		"multiple_fields": {
			ConfigStruct: &struct{ EnvVarA, EnvVarB string }{},
			EnvVarName:   "ENV_VAR_A",
			EnvVarValue:  "asdf",
			Expected:     &struct{ EnvVarA, EnvVarB string }{EnvVarA: "asdf", EnvVarB: ""},
		},
		"golang_camel_case_naming": {
			ConfigStruct: &struct{ JSONFilePath string }{},
			EnvVarName:   "JSON_FILE_PATH",
			EnvVarValue:  "/path/to/file",
			Expected:     &struct{ JSONFilePath string }{JSONFilePath: "/path/to/file"},
		},
		"nested_struct_field": {
			ConfigStruct: &struct {
				Foo string
				Bar *struct {
					Hello   string
					Goodbye int
				}
			}{},
			EnvVarName:  "BAR_GOODBYE",
			EnvVarValue: "8",
			Expected: &struct {
				Foo string
				Bar *struct {
					Hello   string
					Goodbye int
				}
			}{
				Foo: "",
				Bar: &struct {
					Hello   string
					Goodbye int
				}{Hello: "", Goodbye: 8},
			},
		},
		"embedded_field": {
			ConfigStruct: &struct {
				Hello string
				Embed
			}{},
			EnvVarName:  "FOO", // Embed struct had field Foo
			EnvVarValue: "8",
			Expected: &struct {
				Hello string
				Embed
			}{
				Embed: Embed{
					Foo: 8,
				},
			},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(testCase.EnvVarName, testCase.EnvVarValue)
			defer os.Unsetenv(testCase.EnvVarName)
			d, err := dials.Config(context.Background(), testCase.ConfigStruct, &testCase.Source)
			if testCase.ExpectedErr != "" {
				require.Contains(t, err.Error(), testCase.ExpectedErr)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, testCase.Expected, d.View())
			}
		})
	}
}
