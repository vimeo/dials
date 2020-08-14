package parse

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

// splitStringsSlice splits up a string composed of comma-separated values.
// Destination type determined by func passed in.
func splitStringsSlice(s string, addVal func(val string) error) error {
	if len(s) == 0 {
		return nil
	}
	errs := map[scanner.Position]string{}

	inValue := true

	// initialize the scanner (note: sc.Init() blindly overwrites fields with sane-defaults)
	sc := scanner.Scanner{}
	sc.Init(strings.NewReader(s))
	// We need a different default for Mode and Error, though
	sc.Mode = scanner.ScanStrings | scanner.ScanRawStrings |
		scanner.ScanIdents | scanner.ScanChars
	sc.Error = func(s *scanner.Scanner, msg string) {
		errs[s.Pos()] = msg
	}
	sc.IsIdentRune = func(ch rune, i int) bool {
		switch ch {
		case '\\', ',', '"', '\'', '`', '\000':
			return false
		case '.', ':', '/', '+', '-', '$', '%':
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
	for tok := sc.Scan(); tok != scanner.EOF && sc.ErrorCount == 0; tok = sc.Scan() {
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
			if inValue {
				if addErr := addVal(txt); addErr != nil {
					return fmt.Errorf("failed to add val %q: %s",
						txt, addErr)
				}
			} else {
				return fmt.Errorf("unexpected string literal: %s",
					sc.TokenText())
			}
			inValue = false
		case ',':
			inValue = true
		default:
			return fmt.Errorf("unexpected token %s with value: %q",
				scanner.TokenString(tok), sc.TokenText())
		}
	}

	if sc.ErrorCount != 0 {
		return fmt.Errorf("parsing failed: %v", errs)
	}

	return nil
}
