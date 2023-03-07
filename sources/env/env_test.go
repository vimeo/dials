package env

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
)

func testSafeDialsRet[T any](d *dials.Dials[T], err error) (any, error) {
	if d == nil {
		return nil, err
	}
	return d.View(), err
}

func TestEnv(t *testing.T) {
	type Embed struct {
		Foo      int
		Bar      bool
		SomeTime time.Duration
	}
	cases := map[string]struct {
		EnvVarName  string
		EnvVarValue string
		Source      Source
		Run         func(ctx context.Context, src *Source) (any, error)
		Expected    any
		ExpectedErr string
	}{
		"string": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_dials_tag": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					EnvVar string `dials:"ENVIRONMENT_VARIABLE"`
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENVIRONMENT_VARIABLE",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_env_tag": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					EnvVar string `dialsenv:"ENVIRONMENT_VARIABLE"`
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENVIRONMENT_VARIABLE",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"string_with_dials_and_env_tags": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					EnvVar string `dials:"env-var" dialsenv:"ENV_TWO"`
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_TWO",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"int": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar int }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "123",
			Expected:    &struct{ EnvVar int }{EnvVar: 123},
		},
		"int8": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar int8 }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "123",
			Expected:    &struct{ EnvVar int8 }{EnvVar: int8(123)},
		},
		"int8_overflow": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar int8 }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "999999",
			Expected:    nil,
			ExpectedErr: "overflow of int8 type: 999999",
		},
		"bool": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar bool }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "true",
			Expected:    &struct{ EnvVar bool }{EnvVar: true},
		},
		"float32": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar float32 }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "1.123",
			Expected:    &struct{ EnvVar float32 }{EnvVar: float32(1.123)},
		},
		"string_slice": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar []string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: `"a","b"`,
			Expected:    &struct{ EnvVar []string }{EnvVar: []string{"a", "b"}},
		},
		"prefixed_string": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "PREFIX_ENV_VAR",
			EnvVarValue: "asdf",
			Source:      Source{Prefix: "PREFIX"},
			Expected:    &struct{ EnvVar string }{EnvVar: "asdf"},
		},
		"zero_value_string": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "NOT_STRUCT_FIELD_NAME",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVar string }{EnvVar: ""},
		},
		"zero_value_int": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar int }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "NOT_STRUCT_FIELD_NAME",
			EnvVarValue: "123",
			Expected:    &struct{ EnvVar int }{EnvVar: 0},
		},
		"zero_value_bool": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar bool }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "NOT_STRUCT_FIELD_NAME",
			EnvVarValue: "true",
			Expected:    &struct{ EnvVar bool }{EnvVar: false},
		},
		"zero_value_string_slice": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVar []string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR",
			EnvVarValue: "",
			Expected:    &struct{ EnvVar []string }{EnvVar: []string{}},
		},
		"multiple_fields": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ EnvVarA, EnvVarB string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "ENV_VAR_A",
			EnvVarValue: "asdf",
			Expected:    &struct{ EnvVarA, EnvVarB string }{EnvVarA: "asdf", EnvVarB: ""},
		},
		"golang_camel_case_naming": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct{ JSONFilePath string }{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "JSON_FILE_PATH",
			EnvVarValue: "/path/to/file",
			Expected:    &struct{ JSONFilePath string }{JSONFilePath: "/path/to/file"},
		},
		"nested_struct_field": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					Foo string
					Bar *struct {
						Hello   string
						Goodbye int
					}
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
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
		"nested_struct_field_with_slice": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					Foo string
					Bar []struct {
						Hello   string
						Goodbye int
					}
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
			EnvVarName:  "BAR_GOODBYE",
			EnvVarValue: "8",
			Expected: &struct {
				Foo string
				Bar []struct {
					Hello   string
					Goodbye int
				}
			}{
				Foo: "",
				Bar: []struct {
					Hello   string
					Goodbye int
				}(nil),
			},
		},
		"embedded_field": {
			Run: func(ctx context.Context, src *Source) (any, error) {
				cfg := struct {
					Hello string
					Embed
				}{}
				return testSafeDialsRet(dials.Config(context.Background(), &cfg, src))
			},
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

	ctx := context.Background()
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(testCase.EnvVarName, testCase.EnvVarValue)
			defer os.Unsetenv(testCase.EnvVarName)
			cfg, err := testCase.Run(ctx, &testCase.Source)
			if testCase.ExpectedErr != "" {
				require.Contains(t, err.Error(), testCase.ExpectedErr)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, testCase.Expected, cfg)
			}
		})
	}
}
