package caseconversion

import (
	"fmt"
	"go/token"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// GoCaseConverter is a case converter that can decode and encode Go-style
// identifiers.
type GoCaseConverter struct {
	initialisms  []string
	atoms        []string
	loweredAtoms []string
}

// NewGoCaseConverter creates a new GoCaseConverter.  Please see the
// `AddInitialism`, `SetInitialisms`, and `SetAtoms` methods for details on how
// to customize the encoding and decoding behavior.
func NewGoCaseConverter() *GoCaseConverter {
	return &GoCaseConverter{
		initialisms: commonInitialisms,
	}
}

var defaultGoCaseConverter = NewGoCaseConverter()

// SetInitialisms replaces the set of initialisms used by the GoCaseConverter
// with the argument.  Attempting to set an initialism less than two characters
// long will cause a panic.  Also, any duplicates in the list of initialisms
// will be silently discarded.
func (g *GoCaseConverter) SetInitialisms(initialisms []string) {
	g.initialisms = uniqueSortedStrings(initialisms)
	for _, initialism := range g.initialisms {
		if len(initialism) < 2 {
			panic(fmt.Sprintf("initialisms must be at least two characters long; %q is not valid", initialism))
		}
	}
}

// returns a sorted list of unique strings
func uniqueSortedStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	unique := make([]string, 0, len(s))
	for _, str := range s {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			unique = append(unique, str)
		}
	}
	sort.Strings(unique)
	return unique
}

// AddInitialisms adds the passed initialisms to the set of initialisms.
// Attempting to add an initialism less than two characters long will cause a
// panic.  If any of the added initialisms duplicate any existing initialism,
// the duplicates will be silently ignored.
func (g *GoCaseConverter) AddInitialism(initialism ...string) {
	g.SetInitialisms(append(g.initialisms, initialism...))
}

// SetAtoms sets the atoms used by the GoCaseConverter with the argument.
// If any atoms were previously set, SetAtoms replaces them. The set of atoms is
// initially empty, unlike the set of initialisms.
// Atoms will specifically not be split at word boundaries and should
// be provided in the exported-name format as in "ABTest".  Attemping to add an
// atom less than two characters in length will cause a panic.  Also, any
// duplicates in the list of atoms will be removed.
func (g *GoCaseConverter) SetAtoms(atoms []string) {
	g.atoms = uniqueSortedStrings(atoms)

	g.loweredAtoms = make([]string, len(g.atoms))
	for i, atom := range g.atoms {
		if len(atom) < 2 {
			panic(fmt.Sprintf("atoms must be at least two characters long; %q is not valid", atom))
		}
		g.loweredAtoms[i] = strings.ToLower(atom)
	}
}

// Decode implements DecodeCasingFunc for Go-style identifiers.  It consults the
// internal list of initialisms and atoms to determine how to split the string.
func (g *GoCaseConverter) Decode(s string) (DecodedIdentifier, error) {
	if !token.IsIdentifier(s) {
		return nil, fmt.Errorf("only characters of the Letter category or '_' can appear in strings")
	}
	return g.decodeGoCamelCase(s, func(r rune) bool {
		return r == '_'
	})
}

// DecodeGoTags decodes CamelCase, snake_case, and kebab-case strings with fully
// capitalized acronyms into a slice of lower cased strings.
func (g *GoCaseConverter) DecodeGoTags(s string) (DecodedIdentifier, error) {
	return g.decodeGoCamelCase(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
}

// decodeGoCamelCase splits up a string in a slice of lower cased sub-string by
// splitting after fully capitalized acronyms and after the characters that
// signal word boundaries as specified in the passed isWordBoundary function
func (g *GoCaseConverter) decodeGoCamelCase(s string, isWordBoundary func(rune) bool) (DecodedIdentifier, error) {
	words := []string{}

	buf := strings.Builder{}

	sRunes := []rune(s)
	for i := 0; i < len(sRunes); i++ {
		char := sRunes[i]
		if buf.Len() > 0 && (firstCharOfInitialism(s, i) || firstCharAfterInitialism(s, i) || isWordBoundary(char)) {
			// We think we're at a word boundary, but we need to check if this is a prefix for an atom.
			// We're looking for the longest matching atom, so this requires some iteration.
			offset := sort.SearchStrings(g.atoms, buf.String())
			bestMatch := -1
			bestMatchLenDiff := 0
			for ; offset < len(g.atoms) && strings.HasPrefix(g.atoms[offset], buf.String()); offset++ {
				str := buf.String()
				candidate := g.atoms[offset]

				lenDiff := len(candidate) - len(str)
				if lenDiff > 0 && len(str) < (i+lenDiff) {
					// pull off more characters to match the length of the atom
					str += string(sRunes[i : i+lenDiff])
				}

				if strings.EqualFold(candidate, str) {
					// we found an atom that matches exactly, so we should hold
					// on to that before we look for something potentially
					// better
					bestMatch = offset
					bestMatchLenDiff = lenDiff
				}
			}

			if bestMatch >= 0 {
				// we found a match with an atom, so advance the pointer
				words = append(words, g.atoms[bestMatch])
				buf.Reset()
				i += bestMatchLenDiff - 1
				continue
			}

			words = append(words, buf.String())
			buf.Reset()

			if isWordBoundary(char) {
				// if we're on a word boundary, just advance past it
				continue
			}
		}
		buf.WriteRune(char)
	}

	if buf.Len() > 0 {
		// write whatever is left over in the buffer
		words = append(words, buf.String())
	}

	lowerCased := make([]string, 0, len(words))

	// see if any of the initialisms are actually a combination of two ("JSONAPI" or something...)
	for _, word := range words {
		if strings.ToUpper(word) != word {
			// it's not an initialism because it's not all uppercase
			lowerCased = append(lowerCased, strings.ToLower(word))
			continue
		}

		offset := sort.SearchStrings(g.initialisms, word)
		// offset is the position where we would insert this new word, so we
		// should check the word before it to see if it's a prefix (or possibly
		// an exact match)
		for offset > 0 && strings.HasPrefix(word, g.initialisms[offset-1]) {
			lowerCased = append(lowerCased, strings.ToLower(word[:len(g.initialisms[offset-1])]))
			word = word[len(g.initialisms[offset-1]):]
			offset = sort.SearchStrings(g.initialisms, word)
		}

		// if there's anything left to the word, add it
		if len(word) > 0 {
			lowerCased = append(lowerCased, strings.ToLower(word))
		}
	}
	return lowerCased, nil
}

// Encode implements a EncodeCasingFunc for Go-style identifiers and returns an
// exported-style name (with an initial uppercase character).
func (g *GoCaseConverter) Encode(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words))
	for _, w := range words {
		maybeInitialism := strings.ToUpper(w)
		initialismOffset := sort.SearchStrings(g.initialisms, maybeInitialism)

		atomOffset := sort.SearchStrings(g.loweredAtoms, w)

		// check first if it's an atom, then if it's an initialism, and then
		// assume it's just a normal name.
		if atomOffset < len(g.atoms) && g.loweredAtoms[atomOffset] == w {
			b.WriteString(g.atoms[atomOffset])
		} else if initialismOffset < len(g.initialisms) && g.initialisms[initialismOffset] == maybeInitialism {
			b.WriteString(maybeInitialism)
		} else {
			b.WriteString(cases.Title(language.English, cases.NoLower).String(w))
		}
	}
	return b.String()
}

// EncodeUnexported is like Encode, but returns an unexported name (with an
// initial lowercase character) still adhering to the rules of initialisms and
// atoms.
func (g *GoCaseConverter) EncodeUnexported(words DecodedIdentifier) string {
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words))
	for i, w := range words {
		maybeInitialism := strings.ToUpper(w)
		initialismOffset := sort.SearchStrings(g.initialisms, maybeInitialism)

		atomOffset := sort.SearchStrings(g.loweredAtoms, w)

		if atomOffset < len(g.atoms) && g.loweredAtoms[atomOffset] == w {
			if i == 0 {
				b.WriteString(g.loweredAtoms[atomOffset])
			} else {
				b.WriteString(g.atoms[atomOffset])
			}
		} else {
			if i == 0 {
				b.WriteString(w)
			} else if initialismOffset < len(g.initialisms) && g.initialisms[initialismOffset] == maybeInitialism {
				b.WriteString(maybeInitialism)
			} else {
				b.WriteString(cases.Title(language.English, cases.NoLower).String(w))
			}
		}
	}
	return b.String()
}

func decodeCamelCase(typeName, s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("converting case of %q: %s strings can't start with characters of the Decimal Digit category", s, typeName)
	}
	words := []string{}
	lastBoundary := 0
	for z, char := range s {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return nil, fmt.Errorf("converting case of %q: Only characters of the Letter and Decimal Digit categories can appear in %s strings: %c at byte offset %d",
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
		return nil, fmt.Errorf("converting case of %q: First character of upperCamelCase string must be an uppercase character of the Letter category", s)
	}
	return decodeCamelCase("UpperCamelCase", s)
}

// DecodeLowerCamelCase decodes lowerCamelCase strings into a slice of lower-cased sub-strings
func DecodeLowerCamelCase(s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) || !unicode.IsLower(r) {
		return nil, fmt.Errorf("converting case of %q: First character of lowerCamelCase string must be a lowercase character of the Letter category", s)
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

// DecodeGoCamelCase decodes UpperCamelCase and lowerCamelCase strings with
// fully capitalized acronyms (e.g., "jsonAPIDocs") into a slice of lower-cased
// sub-strings.
func DecodeGoCamelCase(s string) (DecodedIdentifier, error) {
	if !token.IsIdentifier(s) {
		return nil, fmt.Errorf("only characters of the Letter category or '_' can appear in strings")
	}
	return defaultGoCaseConverter.Decode(s)
}

// DecodeGoTags decodes CamelCase, snake_case, and kebab-case strings with fully
// capitalized acronyms into a slice of lower cased strings.
func DecodeGoTags(s string) (DecodedIdentifier, error) {
	return defaultGoCaseConverter.DecodeGoTags(s)
}

// List from https://github.com/golang/lint/blob/master/lint.go
var commonInitialisms = []string{"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS"}

func decodeLowerCaseWithSplitChar(splitChar rune, typeName, s string) (DecodedIdentifier, error) {
	// ignore the size of the rune
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsDigit(r) {
		return nil, fmt.Errorf("converting case of %q: %s strings can't start with characters of the Decimal Digit category", s, typeName)
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
			return nil, fmt.Errorf("converting case of %q: Only lower-case letter-category characters, digits, and '%c' can appear in `%s` strings: %c at byte-offset %d does not comply", s, splitChar, typeName, char, z)
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
		return nil, fmt.Errorf("converting case of %q: UPPER_SNAKE_CASE strings can't start with characters of the Decimal Digit category", s)
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
			return nil, fmt.Errorf("converting case of %q: Only uppercase characters of the Letter category and '_' can appear in UPPER_SNAKE_CASE strings: %c in at byte-offset %d does not comply", s, char, z)
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
		return nil, fmt.Errorf("converting case of %q: Case_Preserving_Snake_Case strings can't start with characters of the Decimal Digit category", s)

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
			return nil, fmt.Errorf("converting case of %q: Only characters of the Letter category and '_' can appear in Preserving_Snake_Case strings: %c at byte-offset %d does not comply", s, char, z)
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
		b.WriteString(cases.Title(language.English, cases.NoLower).String(w))
	}
	return b.String()
}

// EncodeLowerCamelCase encodes a slice of words into lowerCamelCase
func EncodeLowerCamelCase(words DecodedIdentifier) string {
	if len(words) == 0 {
		return ""
	}
	b := strings.Builder{}
	b.Grow(aggregateStringLen(words))
	b.WriteString(words[0])
	for _, w := range words[1:] {
		b.WriteString(cases.Title(language.English, cases.NoLower).String(w))
	}
	return b.String()
}

// EncodeKebabCase encodes a slice of words into kebab-case
func EncodeKebabCase(words DecodedIdentifier) string {
	return strings.Join(words, "-")
}

// EncodeLowerSnakeCase encodes a slice of words into lower_snake_case
func EncodeLowerSnakeCase(words DecodedIdentifier) string {
	if len(words) == 0 {
		return ""
	}
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
	if len(words) == 0 {
		return ""
	}
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
