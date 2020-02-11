package parsestring

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
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
		scanner.ScanIdents | scanner.ScanInts |
		scanner.ScanFloats | scanner.ScanChars
	sc.Error = func(s *scanner.Scanner, msg string) {
		errs[s.Pos()] = msg
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
