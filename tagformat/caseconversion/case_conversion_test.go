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

	{"jsonAPI", []string{"json", "api"}, DecodeGoTags, false},
	{"value-3", []string{"value", "3"}, DecodeGoTags, false},
	{"decode_golangCamelCase_try_", []string{"decode", "golang", "camel", "case", "try"}, DecodeGoTags, false},
	{"AB_Test-something_fun", []string{"ab", "test", "something", "fun"}, DecodeGoTags, false},
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
	{[]string{}, "", EncodeUpperCamelCase},
	{[]string{"lower", "camel", "case"}, "lowerCamelCase", EncodeLowerCamelCase},
	{[]string{}, "", EncodeLowerCamelCase},
	{[]string{"kebab", "case", "string"}, "kebab-case-string", EncodeKebabCase},
	{[]string{}, "", EncodeKebabCase},
	{[]string{"loweR", "SNAKE", "Case"}, "lower_snake_case", EncodeLowerSnakeCase},
	{[]string{}, "", EncodeLowerSnakeCase},
	{[]string{"upper", "snake", "case"}, "UPPER_SNAKE_CASE", EncodeUpperSnakeCase},
	{[]string{}, "", EncodeUpperSnakeCase},
	{[]string{"case", "PRESERVING", "Snake"}, "case_PRESERVING_Snake", EncodeCasePreservingSnakeCase},
	{[]string{}, "", EncodeCasePreservingSnakeCase},
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

func TestGoCaseConverter(t *testing.T) {
	g := NewGoCaseConverter()
	g.SetAtoms([]string{"RaNsoMNoTe", "ZZTop", "ABTests", "ABTest"})
	g.AddInitialism("XSL", "XSLFO")

	for testName, tbl := range map[string]struct {
		original   string
		exported   string
		unexported string
		expected   []string
	}{
		"InitialismWithSubstring": {
			original:   "XSLFormatter",
			unexported: "xslFormatter",
			expected:   []string{"xsl", "formatter"},
		},
		"TwoInitialisms": {
			original:   "JSONAPI",
			unexported: "jsonAPI",
			expected:   []string{"json", "api"},
		},
		"SingleInitialism": {
			original:   "API",
			unexported: "api",
			expected:   []string{"api"},
		},
		"NotKnownInitialism": {
			original:   "XSDFile",
			exported:   "XsdFile",
			unexported: "xsdFile",
			expected:   []string{"xsd", "file"},
		},
		"TwoInitialismsWithSuffix": {
			original:   "JSONAPIA",
			unexported: "jsonAPIA",
			expected:   []string{"json", "api", "a"},
		},
		"ThreeInitialisms": {
			original:   "XMLJSONAPI",
			unexported: "xmlJSONAPI",
			expected:   []string{"xml", "json", "api"},
		},
		"TestLongConcatted": {
			original:   "TestJSONAPI",
			unexported: "testJSONAPI",
			expected:   []string{"test", "json", "api"},
		},
		"TestLongConcattedWithSuffix": {
			original:   "TestJSONAPIAddress",
			unexported: "testJSONAPIAddress",
			expected:   []string{"test", "json", "api", "address"},
		},
		"TestAtomAlone": {
			original:   "ABTest",
			unexported: "abtest",
			expected:   []string{"abtest"},
		},
		"TestAtomLongerString": {
			original:   "ABTestsGroup",
			unexported: "abtestsGroup",
			expected:   []string{"abtests", "group"},
		},
		"TestAtomWithInitialismSuffix": {
			original:   "ABTestID",
			unexported: "abtestID",
			expected:   []string{"abtest", "id"},
		},
		"TestAtomWithPrefix": {
			original:   "TheRaNsoMNoTe",
			unexported: "theRaNsoMNoTe",
			expected:   []string{"the", "ransomnote"},
		},
		"TwoAtoms": {
			original:   "ABTestZZTop",
			unexported: "abtestZZTop",
			expected:   []string{"abtest", "zztop"},
		},
	} {
		t.Run(testName, func(t *testing.T) {
			words, err := g.Decode(tbl.original)
			require.NoError(t, err)
			assert.Equal(t, DecodedIdentifier(tbl.expected), words)

			if tbl.exported == "" {
				tbl.exported = tbl.original
			}
			encoded := g.Encode(DecodedIdentifier(tbl.expected))
			assert.Equal(t, tbl.exported, encoded)

			encodedUnexported := g.EncodeUnexported(DecodedIdentifier(tbl.expected))
			assert.Equal(t, tbl.unexported, encodedUnexported)
		})
	}

	t.Run("TestInitialismPanic", func(t *testing.T) {
		assert.Panics(t, func() {
			g := NewGoCaseConverter()
			g.SetInitialisms([]string{"A"})
		})
	})

	t.Run("TestAtomPanic", func(t *testing.T) {
		assert.Panics(t, func() {
			g := NewGoCaseConverter()
			g.SetAtoms([]string{"A"})
		})
	})
}
