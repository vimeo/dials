package caseconversion

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DecodeCasingFunc takes in an identifier in a case such as camelCase or
// snake_case and splits it up into a DecodedIdentifier for encoding by an
// EncodeCasingFunc into a different case.
type DecodeCasingFunc func(string) (DecodedIdentifier, error)

// EncodeCasingFunc combines the contents of a DecodedIdentifier into an
// identifier in a case such as camelCase or snake_case.
type EncodeCasingFunc func(DecodedIdentifier) string

// DecodedIdentifier is an slice of lowercase words (e.g., []string{"test",
// "string"}) produced by a DecodeCasingFunc, which can be encoded by an
// EncodeCasingFunc into a string in the specified case (e.g., with
// EncodeLowerCamelCase, "testString").
type DecodedIdentifier []string

func decodeCamelCase(typeName, s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("Converting case of %q: %s strings can't start with characters of the Decimal Digit category", s, typeName)
	}
	words := []string{}
	lastBoundary := 0
	for z, char := range s {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return nil, fmt.Errorf("Converting case of %q: Only characters of the Letter and Decimal Digit categories can appear in %s strings: %c at byte offset %d",
				s, typeName, char, z)
		}
		if unicode.IsUpper(char) {
			// flush out current substring
			if lastBoundary < z {
				words = append(words, strings.ToLower(s[lastBoundary:z]))
			}
			lastBoundary = z
		}
	}
	// flush one last time to get the remainder of the string
	words = append(words, strings.ToLower(s[lastBoundary:]))
	return words, nil
}

// DecodeUpperCamelCase decodes UpperCamelCase strings into a slice of lower-cased sub-strings
func DecodeUpperCamelCase(s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) || !unicode.IsUpper(r) {
		return nil, fmt.Errorf("Converting case of %q: First character of upperCamelCase string must be an uppercase character of the Letter category", s)
	}
	return decodeCamelCase("UpperCamelCase", s)
}

// DecodeLowerCamelCase decodes lowerCamelCase strings into a slice of lower-cased sub-strings
func DecodeLowerCamelCase(s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) || !unicode.IsLower(r) {
		return nil, fmt.Errorf("Converting case of %q: First character of lowerCamelCase string must be a lowercase character of the Letter category", s)
	}
	return decodeCamelCase("lowerCamelCase", s)
}

// firstCharOfInitialism, as used in DecodeGoCamelCase, attempts to
// detect when the indexed rune is the first character of an initialism (e.g.,
// json*A*PI).
func firstCharOfInitialism(s string, i int) bool {
	r1, rl1 := utf8.DecodeRuneInString(s[i:])

	// ignore the rune length for the previous character
	r2, _ := utf8.DecodeLastRuneInString(s[:i])

	// need the equal to for when the rune is the last char in the string (ex: EnvVarA)
	return len(s) >= i+rl1 && i >= 1 && unicode.IsUpper(r1) && unicode.IsLower(r2)
}

// firstCharAfterInitialism, as used in DecodeGoCamelCase, attempts to
// detect when the indexed rune is the first character of a non-initialism after
// an initialism (e.g., JSON*F*ile).
func firstCharAfterInitialism(s string, i int) bool {
	r1, rl1 := utf8.DecodeRuneInString(s[i:])
	// ensure the rune isn't the last character of the string
	if i+rl1 >= len(s) {
		return false
	}
	r2, rl2 := utf8.DecodeRuneInString(s[i+rl1:])
	return i+rl1+rl2 < len(s) && unicode.IsUpper(r1) && unicode.IsLower(r2)
}

// lastCharOfInitialismAtEOS, as used in DecodeGoCamelCase, attempts to
// detect when the indexed rune is the last character of an initialism at the
// end of a string (e.g., jsonAP*I*).
func lastCharOfInitialismAtEOS(s string, i int) bool {
	s1 := s[i:]
	r, rl := utf8.DecodeRuneInString(s1)

	return i+rl == len(s) && unicode.IsUpper(r)
}

// decodeGoCamelCase splits up a string in a slice of lower cased sub-string by
// splitting after fully capitalized acronyms and after the characters that
// signal word boundaries as specified in the passed isWordBoundary function
func decodeGoCamelCase(s string, isWordBoundary func(rune) bool) (DecodedIdentifier, error) {
	words := []string{}
	lastBoundary := 0
	for i, char := range s {
		if firstCharOfInitialism(s, i) || firstCharAfterInitialism(s, i) || isWordBoundary(char) {
			if lastBoundary < i {
				word := s[lastBoundary:i]
				if word == strings.ToUpper(word) {
					words = append(words, extractInitialisms(word)...)
				} else {
					words = append(words, strings.ToLower(word))
				}
			}
			switch {
			case isWordBoundary(char):
				lastBoundary = i + 1
			default:
				lastBoundary = i
			}

		} else if lastCharOfInitialismAtEOS(s, i) {
			if lastBoundary < i {
				word := s[lastBoundary:]
				if word == strings.ToUpper(word) {
					words = append(words, extractInitialisms(word)...)
					return words, nil
				}
			}
			lastBoundary = i
		}
	}

	if last := strings.ToLower(s[lastBoundary:]); len(last) > 0 {
		words = append(words, strings.ToLower(s[lastBoundary:]))
	}

	return words, nil
}

// TODO: Add EncodeGoCamelCase function and set as default name encoder in
// FlattenMangler

// DecodeGoCamelCase decodes UpperCamelCase and lowerCamelCase strings with
// fully capitalized acronyms (e.g., "jsonAPIDocs") into a slice of lower-cased
// sub-strings.
func DecodeGoCamelCase(s string) (DecodedIdentifier, error) {
	if !token.IsIdentifier(s) {
		return nil, fmt.Errorf("Only characters of the Letter category or '_' can appear in strings")
	}
	return decodeGoCamelCase(s, func(r rune) bool {
		return r == '_'
	})
}

// DecodeGoTags decodes CamelCase, snake_case, and kebab-case strings with fully
// capitalized acronyms into a slice of lower cased strings.
func DecodeGoTags(s string) (DecodedIdentifier, error) {
	return decodeGoCamelCase(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
}

// List from https://github.com/golang/lint/blob/master/lint.go
var commonInitialisms = []string{"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS"}

// Given an entirely uppercase string, extract any initialisms sequentially from the start of the string and return them with the remainder of the string
func extractInitialisms(s string) []string {
	words := []string{}

	for {
		initialismFound := false
		for _, initialism := range commonInitialisms {
			if len(s) >= len(initialism) && initialism == s[:len(initialism)] {
				initialismFound = true
				words = append(words, strings.ToLower(initialism))
				s = s[len(initialism):]
			}
		}
		if !initialismFound {
			break
		}
	}

	if len(s) > 0 {
		words = append(words, strings.ToLower(s))
	}

	return words
}

func decodeLowerCaseWithSplitChar(splitChar rune, typeName, s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("Converting case of %q: %s strings can't start with characters of the Decimal Digit category", s, typeName)
	}

	words := []string{}
	lastBoundary := 0
	for z, char := range s {
		if char == splitChar {
			// flush
			if lastBoundary < z {
				words = append(words, s[lastBoundary:z])
			}
			lastBoundary = z + utf8.RuneLen(splitChar)
		} else if (!unicode.IsLetter(char) && !unicode.IsDigit(char)) || (!unicode.IsLower(char) && !unicode.IsDigit(char)) {
			return nil, fmt.Errorf("Converting case of %q: Only lower-case letter-category characters, digits, and '%c' can appear in `%s` strings: %c at byte-offset %d does not comply", s, splitChar, typeName, char, z)
		}
	}
	// flush one last time to get the remainder of the string
	if last := strings.ToLower(s[lastBoundary:]); len(last) > 0 {
		words = append(words, strings.ToLower(s[lastBoundary:]))
	}
	return words, nil
}

// DecodeLowerSnakeCase decodes lower_snake_case into a slice of lower-cased sub-strings
func DecodeLowerSnakeCase(s string) (DecodedIdentifier, error) {
	return decodeLowerCaseWithSplitChar('_', "lower_snake_case", s)
}

// DecodeKebabCase decodes kebab-case into a slice of lower-cased sub-strings
func DecodeKebabCase(s string) (DecodedIdentifier, error) {
	return decodeLowerCaseWithSplitChar('-', "kebab-case", s)
}

// DecodeUpperSnakeCase decodes UPPER_SNAKE_CASE (sometimes called
// SCREAMING_SNAKE_CASE) into a slice of lower-cased sub-strings
func DecodeUpperSnakeCase(s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("Converting case of %q: UPPER_SNAKE_CASE strings can't start with characters of the Decimal Digit category", s)
	}

	words := []string{}
	lastBoundary := 0
	for z, char := range s {
		if char == '_' {
			// flush
			if lastBoundary < z {
				words = append(words, strings.ToLower(s[lastBoundary:z]))
			}
			lastBoundary = z + 1
		} else if (!unicode.IsLetter(char) && !unicode.IsDigit(char)) || (!unicode.IsUpper(char) && !unicode.IsDigit(char)) {
			return nil, fmt.Errorf("Converting case of %q: Only uppercase characters of the Letter category and '_' can appear in UPPER_SNAKE_CASE strings: %c in at byte-offset %d does not comply", s, char, z)
		}
	}
	// flush one last time to get the remainder of the string
	if last := strings.ToLower(s[lastBoundary:]); len(last) > 0 {
		words = append(words, strings.ToLower(s[lastBoundary:]))
	}
	return words, nil
}

// DecodeCasePreservingSnakeCase decodes Case_Preserving_Snake_Case into a
// slice of lower-cased sub-string
func DecodeCasePreservingSnakeCase(s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("Converting case of %q: Case_Preserving_Snake_Case strings can't start with characters of the Decimal Digit category", s)

	}
	words := []string{}
	lastBoundary := 0

	for z, char := range s {
		if char == '_' {
			// flush
			if lastBoundary < z {
				words = append(words, strings.ToLower(s[lastBoundary:z]))
			}
			lastBoundary = z + 1
		} else if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return nil, fmt.Errorf("Converting case of %q: Only characters of the Letter category and '_' can appear in Preserving_Snake_Case strings: %c at byte-offset %d does not comply", s, char, z)
		}
	}
	words = append(words, strings.ToLower(s[lastBoundary:]))
	return words, nil
}

func aggregateStringLen(words DecodedIdentifier) int {
	total := 0
	for _, w := range words {
		total += len(w)
	}
	return total
}

// EncodeUpperCamelCase encodes a slice of words into UpperCamelCase
func EncodeUpperCamelCase(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words))
	for _, w := range words {
		b.WriteString(strings.Title(w))
	}
	return b.String()
}

// EncodeLowerCamelCase encodes a slice of words into lowerCamelCase
func EncodeLowerCamelCase(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words))
	b.WriteString(words[0])
	for _, w := range words[1:] {
		b.WriteString(strings.Title(w))
	}
	return b.String()
}

// EncodeKebabCase encodes a slice of words into kebab-case
func EncodeKebabCase(words DecodedIdentifier) string {
	return strings.Join(words, "-")
}

// EncodeLowerSnakeCase encodes a slice of words into lower_snake_case
func EncodeLowerSnakeCase(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words) + len(words) - 1)
	for i, w := range words {
		b.WriteString(strings.ToLower(w))
		if i != len(words)-1 {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// EncodeUpperSnakeCase encodes a slice of words into UPPER_SNAKE_CASE (AKA
// SCREAMING_SNAKE_CASE)
func EncodeUpperSnakeCase(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words) + len(words) - 1)
	for i, w := range words {
		b.WriteString(strings.ToUpper(w))
		if i != len(words)-1 {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// EncodeCasePreservingSnakeCase encodes a slice of words into case_Preserving_snake_case
func EncodeCasePreservingSnakeCase(words DecodedIdentifier) string {
	return strings.Join(words, "_")
}
