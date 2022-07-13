package flaghelper

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/vimeo/dials/parse"
)

// StringSliceFlag is a wrapper around a string slice
type StringSliceFlag struct {
	s *[]string
}

// NewStringSliceFlag is a constructor for StringSliceFlag
func NewStringSliceFlag(s *[]string) *StringSliceFlag {
	return &StringSliceFlag{s: s}
}

// Set implement pflag.Value and flag.Value
func (v *StringSliceFlag) Set(s string) error {
	parsed, err := parse.StringSlice(s)
	if err != nil {
		return err
	}
	v.s = &parsed
	return nil
}

// Get implements flag.Value
func (v *StringSliceFlag) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *StringSliceFlag) String() string {
	if v.s == nil {
		return ""
	}
	b := strings.Builder{}
	for i, z := range *v.s {
		b.WriteString(strconv.Quote(z))
		if i < len(*v.s)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

// StringSetFlag is a wrapper around map[string]struct used for implementing sets
type StringSetFlag struct {
	s *map[string]struct{}
}

// NewStringSetFlag is the constructor for StringSetFlags
func NewStringSetFlag(m *map[string]struct{}) *StringSetFlag {
	return &StringSetFlag{s: m}
}

// Set implement pflag.Value and flag.Value
func (v *StringSetFlag) Set(s string) error {
	parsed, err := parse.StringSet(s)
	if err != nil {
		return err
	}
	*v.s = parsed
	return nil
}

// Get implements flag.Value
func (v *StringSetFlag) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *StringSetFlag) String() string {
	if v.s == nil {
		return ""
	}
	slc := make([]string, 0, len(*v.s))
	for val := range *v.s {
		slc = append(slc, val)
	}

	// sort strings so we get a stable output
	sort.Strings(slc)

	b := strings.Builder{}
	for i, z := range slc {
		b.WriteString(strconv.Quote(z))
		if i < len(*v.s)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

// Type implements pflag.Value
func (v *StringSetFlag) Type() string {
	return fmt.Sprintf("%T", v.s)
}

// MapStringStringSliceFlag is a wrapper around map[string][]string
type MapStringStringSliceFlag struct {
	s *map[string][]string
}

// NewMapStringStringSliceFlag is the constructor for MapStringStringSliceFlag
func NewMapStringStringSliceFlag(m *map[string][]string) *MapStringStringSliceFlag {
	return &MapStringStringSliceFlag{s: m}
}

// Set implement pflag.Value and flag.Value
func (v *MapStringStringSliceFlag) Set(s string) error {
	parsed, err := parse.StringStringSliceMap(s)
	if err != nil {
		return err
	}
	*v.s = parsed
	return nil
}

// Get implements flag.Value
func (v *MapStringStringSliceFlag) Get() interface{} {
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *MapStringStringSliceFlag) String() string {
	if v.s == nil {
		return ""
	}

	keyslc := make([]string, 0, len(*v.s))
	for k := range *v.s {
		keyslc = append(keyslc, k)
	}
	sort.Strings(keyslc)

	b := strings.Builder{}

	for ki, k := range keyslc {
		vs := (*v.s)[k]
		quotedKey := strconv.Quote(k)
		for i, z := range vs {
			b.WriteString(quotedKey)
			b.WriteRune(':')
			b.WriteString(strconv.Quote(z))
			// we want to omit the trailing comma on the last k-v pair.
			if !(ki >= len(*v.s)-1 && i >= len(vs)-1) {
				b.WriteRune(',')
			}
		}
	}
	return b.String()
}

// Type implements pflag.Value
func (v *MapStringStringSliceFlag) Type() string {
	return fmt.Sprintf("%T", v.s)
}

// MapStringStringFlag is a wrapper around *map[string]string
type MapStringStringFlag struct {
	s *map[string]string
}

// NewMapStringStringFlag is the constructor for MapStringStringFlag
func NewMapStringStringFlag(m *map[string]string) *MapStringStringFlag {
	return &MapStringStringFlag{s: m}
}

// Set implement pflag.Value and flag.Value
func (v *MapStringStringFlag) Set(s string) error {
	parsed, err := parse.Map(s, reflect.TypeOf(map[string]string{}))
	if err != nil {
		return err
	}
	castParsed := parsed.Interface().(map[string]string)
	*v.s = castParsed
	return nil
}

// Get implements flag.Value
func (v *MapStringStringFlag) Get() interface{} {
	return *v.s
}

// String implements flag.Value and pflag.Value
func (v *MapStringStringFlag) String() string {
	if v.s == nil || len(*v.s) == 0 {
		return ""
	}

	keyslc := make([]string, 0, len(*v.s))
	for k := range *v.s {
		keyslc = append(keyslc, k)
	}
	sort.Strings(keyslc)

	b := strings.Builder{}

	for i, k := range keyslc {
		val := (*v.s)[k]
		b.WriteString(strconv.Quote(k))
		b.WriteRune(':')
		b.WriteString(strconv.Quote(val))
		if i < len(keyslc)-1 {
			b.WriteRune(',')
		}
	}
	return b.String()
}

// Type implements pflag.Value
func (v *MapStringStringFlag) Type() string {
	return fmt.Sprintf("%T", v.s)
}
