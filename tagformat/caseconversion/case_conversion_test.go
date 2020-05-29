package caseconversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var decodeCases = []struct {
	original    string
	decoded     DecodedIdentifier
	decoderFunc func(string) (DecodedIdentifier, error)
	error       bool
}{
	{"UpperCamelCase", []string{"upper", "camel", "case"}, DecodeUpperCamelCase, false},
	{"lowerCamelCase", []string{"lower", "camel", "case"}, DecodeUpperCamelCase, true},
	{"UpperCamelCaseU", []string{"upper", "camel", "case", "u"}, DecodeUpperCamelCase, false},
	{"Case", []string{"case"}, DecodeUpperCamelCase, false},
	{"UCCase", []string{"u", "c", "case"}, DecodeUpperCamelCase, false},
	{"UC_Case", []string{}, DecodeUpperCamelCase, true},
	{"Upper12CamelCase", []string{"upper12", "camel", "case"}, DecodeUpperCamelCase, false},

	{"lowerCamelCase", []string{"lower", "camel", "case"}, DecodeLowerCamelCase, false},
	{"lowerCamelCaseU", []string{"lower", "camel", "case", "u"}, DecodeLowerCamelCase, false},
	{"lowerCamel0CaseU", []string{"lower", "camel0", "case", "u"}, DecodeLowerCamelCase, false},
	{"NotLowerCamelCase", []string{}, DecodeLowerCamelCase, true},
	{"1errorCase", []string{}, DecodeLowerCamelCase, true},

	{"kebab-case-string", []string{"kebab", "case", "string"}, DecodeKebabCase, false},
	{"1kebab-case-string", []string{}, DecodeKebabCase, true},
	{"kebab1-case-string", []string{"kebab1", "case", "string"}, DecodeKebabCase, false},
	{"kebab1-case-string-", []string{"kebab1", "case", "string"}, DecodeKebabCase, false},
	{"kebab-case-string-u", []string{"kebab", "case", "string", "u"}, DecodeKebabCase, false},

	{"UPPER_SNAKE_CASE", []string{"upper", "snake", "case"}, DecodeUpperSnakeCase, false},
	{"1UPPER_SNAKE_CASE", []string{}, DecodeUpperSnakeCase, true},
	{"UPPER_SNAKE_CASE1", []string{"upper", "snake", "case1"}, DecodeUpperSnakeCase, false},
	{"UPPER_SNAKE_CASE_U", []string{"upper", "snake", "case", "u"}, DecodeUpperSnakeCase, false},
	{"UPPER_SNAKE_CASE_U_", []string{"upper", "snake", "case", "u"}, DecodeUpperSnakeCase, false},

	{"lower_snake_case", []string{"lower", "snake", "case"}, DecodeLowerSnakeCase, false},
	{"lower_snake_case_u", []string{"lower", "snake", "case", "u"}, DecodeLowerSnakeCase, false},
	{"lower__snake_case", []string{"lower", "snake", "case"}, DecodeLowerSnakeCase, false},
	{"_lower_snake_case", []string{"lower", "snake", "case"}, DecodeLowerSnakeCase, false},
	{"lower_snake_case_", []string{"lower", "snake", "case"}, DecodeLowerSnakeCase, false},

	{"caSe_pREserving_sNake_Case", []string{"case", "preserving", "snake", "case"}, DecodeCasePreservingSnakeCase, false},
	// only letters are allowed so "&" in "case" will cause an error
	{"ca&e_pREserving_sNake_Case", []string{}, DecodeCasePreservingSnakeCase, true},

	{"jsonAPI", []string{"json", "api"}, DecodeGoCamelCase, false},
	{"JSONAPI", []string{"json", "api"}, DecodeGoCamelCase, false},
	{"TestJSONAPI", []string{"test", "json", "api"}, DecodeGoCamelCase, false},
	{"TestSOMEJSONAPI", []string{"test", "somejsonapi"}, DecodeGoCamelCase, false},
	{"UpperCamelCase", []string{"upper", "camel", "case"}, DecodeGoCamelCase, false},
	{"lowerCamelCase", []string{"lower", "camel", "case"}, DecodeGoCamelCase, false},
	{"UpperCamelCaseAPI", []string{"upper", "camel", "case", "api"}, DecodeGoCamelCase, false},
	{"UpperCamelCaseAPIDocs", []string{"upper", "camel", "case", "api", "docs"}, DecodeGoCamelCase, false},
	{"UpperCamelCaseXMLAPIDocs", []string{"upper", "camel", "case", "xml", "api", "docs"}, DecodeGoCamelCase, false},
	{"ABTest", []string{"ab", "test"}, DecodeGoCamelCase, false},
	{"jsonABTest", []string{"json", "ab", "test"}, DecodeGoCamelCase, false},
	{"decode_golangCamelCase_try", []string{"decode", "golang", "camel", "case", "try"}, DecodeGoCamelCase, false},
	{"decode_golangCamelCase_try_", []string{"decode", "golang", "camel", "case", "try"}, DecodeGoCamelCase, false},
	{"A", []string{"a"}, DecodeGoCamelCase, false},
	{"EnvVarA", []string{"env", "var", "a"}, DecodeGoCamelCase, false},
}

func TestDecode(t *testing.T) {
	for _, c := range decodeCases {
		decodeCase := c
		t.Run(decodeCase.original, func(t *testing.T) {
			t.Parallel()

			is, err := decodeCase.decoderFunc(decodeCase.original)
			if err != nil && decodeCase.error == true {
				return
			}
			require.NoError(t, err)
			ought := decodeCase.decoded
			assert.Equal(t, ought, is)
		})
	}
}

var encodeCases = []struct {
	decoded     DecodedIdentifier
	encoded     string
	encoderFunc func(DecodedIdentifier) string
}{
	{[]string{"upper", "camel", "case"}, "UpperCamelCase", EncodeUpperCamelCase},
	{[]string{"lower", "camel", "case"}, "lowerCamelCase", EncodeLowerCamelCase},
	{[]string{"kebab", "case", "string"}, "kebab-case-string", EncodeKebabCase},
	{[]string{"loweR", "SNAKE", "Case"}, "lower_snake_case", EncodeLowerSnakeCase},
	{[]string{"upper", "snake", "case"}, "UPPER_SNAKE_CASE", EncodeUpperSnakeCase},
	{[]string{"case", "PRESERVING", "Snake"}, "case_PRESERVING_Snake", EncodeCasePreservingSnakeCase},
}

func TestEncode(t *testing.T) {
	for _, c := range encodeCases {
		encodeCase := c
		t.Run(encodeCase.encoded, func(t *testing.T) {
			t.Parallel()

			is := encodeCase.encoderFunc(encodeCase.decoded)
			ought := encodeCase.encoded
			assert.Equal(t, ought, is)
		})
	}
}

var initialismCases = []struct {
	original string
	returned []string
}{
	{"JSONAPI", []string{"json", "api"}},
	{"JSONAPIA", []string{"json", "api", "a"}},
	{"XMLJSONAPI", []string{"xml", "json", "api"}},
}

func TestExtractInitialisms(t *testing.T) {
	for _, initialismCase := range initialismCases {
		t.Run(initialismCase.original, func(t *testing.T) {
			is := extractInitialisms(initialismCase.original)
			ought := initialismCase.returned
			assert.Equal(t, ought, is)
		})
	}
}
