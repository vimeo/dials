package parse

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

// splitMap splits the values that are csv key:value-pairs of strings in a
// string into a map[string][]string.
func splitMap(s string, addKV func(k, v string) error) error {
	errs := map[scanner.Position]string{}

	// initialize the scanner (note: sc.Init() blindly overwrites fields with sane-defaults)
	sc := scanner.Scanner{}
	sc.Init(strings.NewReader(s))
	// We need different defaults for Mode and Error, though
	sc.Mode = scanner.ScanStrings | scanner.ScanRawStrings |
		scanner.ScanIdents | scanner.ScanInts |
		scanner.ScanFloats | scanner.ScanChars
	sc.Error = func(s *scanner.Scanner, msg string) {
		errs[s.Pos()] = msg
	}
	// Override the IsIdentRune callback. Note that this differs from
	// similar callback in splitStringsSlice() by the presence of colon
	// (`:`) in the disallow-list rather than the allow-list, as maps use
	// colons to separate keys and values, while they have no meaning for
	// string-sets and string-slices.
	sc.IsIdentRune = func(ch rune, i int) bool {
		switch ch {
		case '\\', ',', '"', '\'', '`', '\000', ':':
			return false
		case '.', '/', '+', '-', '$', '%':
			// A few special characters we want to guarantee are
			// caught as allowed
			return true
		default:
		}
		// Disallow whitespace first, then check whether it's printable.
		if (ch < ' ' && ch >= 0) && (sc.Whitespace&(1<<ch) > 0) {
			return false
		}
		// IsPrint includes letter, number, symbol and a couple other
		// character-classes
		if unicode.IsPrint(ch) {
			return true
		}
		return false
	}

	inKey := true
	inValue := false
	curKey := ""
	curVal := ""
	for tok := sc.Scan(); sc.ErrorCount == 0; tok = sc.Scan() {
		switch tok {
		case scanner.String, scanner.RawString, scanner.Ident, scanner.Float, scanner.Int:
			txt := sc.TokenText()
			if tok == scanner.String || tok == scanner.RawString {
				parsedtxt, unquoteErr := strconv.Unquote(sc.TokenText())
				if unquoteErr != nil {
					return fmt.Errorf("unable to unquote string: %s", unquoteErr)
				}
				txt = parsedtxt
			}

			if inKey {
				curKey = txt
			} else if inValue && curKey != "" {
				curVal = txt
			} else {
				return fmt.Errorf("unexpected string literal: %s",
					sc.TokenText())
			}
		case ',':
			if curKey != "" {
				if addErr := addKV(curKey, curVal); addErr != nil {
					return fmt.Errorf("map parsing failed on key %q: %s",
						curKey, addErr)
				}
			}

			curKey = ""
			curVal = ""
			inKey = true
			inValue = false
		case ':':
			if inValue || curKey == "" {
				return fmt.Errorf("unexpected colon")
			}
			inKey = false
			inValue = true
		case scanner.EOF:
			if curKey != "" {
				if addErr := addKV(curKey, curVal); addErr != nil {
					return fmt.Errorf("map parsing failed on key %q: %s",
						curKey, addErr)
				}
			}
			return nil
		}
	}

	if sc.ErrorCount != 0 {
		return fmt.Errorf("parsing failed: %v", errs)
	}

	return nil
}
