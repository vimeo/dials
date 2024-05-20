// Package jsontypes contains helper types used by the JSON and Cue decoders to
// facilitate more natural decoding.
package jsontypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// ParsingDuration implements [encoding/json.Unmarshaler], supporting both
// quoted strings that are parseable with [time.ParseDuration], and integer nanoseconds if it's a number
type ParsingDuration int64

// UnmarshalJSON implements [encoding/json.Unmarshaler] for ParsingDuration.
func (p *ParsingDuration) UnmarshalJSON(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	n, tokErr := d.Token()
	if tokErr != nil {
		return fmt.Errorf("failed to parse token: %w", tokErr)
	}
	switch v := n.(type) {
	case string:
		dur, durParseErr := time.ParseDuration(v)
		if durParseErr != nil {
			return fmt.Errorf("failed to parse %q as duration: %w", v, durParseErr)
		}
		*p = ParsingDuration(dur)
		return nil
	case json.Number:
		i, intParseErr := v.Int64()
		if intParseErr != nil {
			return fmt.Errorf("failed to parse number as integer nanoseconds: %w", intParseErr)
		}
		*p = ParsingDuration(i)
		return nil
	default:
		return fmt.Errorf("unexpected JSON token-type %T; expected string or number", n)
	}
}
