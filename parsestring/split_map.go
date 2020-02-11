package parsestring

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
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
