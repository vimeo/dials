package flag

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/vimeo/dials/parsestring"
)

type stringSliceFlag struct {
	s *[]string
}

func (v *stringSliceFlag) Set(s string) error {
	parsed, err := parsestring.ParseStringSlice(s)
	if err != nil {
		return err
	}
	v.s = &parsed
	return nil
}

func (v *stringSliceFlag) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

func (v *stringSliceFlag) String() string {
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

type stringSetFlag struct {
	s *map[string]struct{}
}

func (v *stringSetFlag) Set(s string) error {
	parsed, err := parsestring.ParseStringSet(s)
	if err != nil {
		return err
	}
	v.s = &parsed
	return nil
}

func (v *stringSetFlag) Get() interface{} {
	if v.s == nil {
		return []string{}
	}
	return *v.s
}

func (v *stringSetFlag) String() string {
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

type mapStringStringSliceFlag struct {
	s *map[string][]string
}

func (v *mapStringStringSliceFlag) Set(s string) error {
	parsed, err := parsestring.ParseStringStringSliceMap(s)
	if err != nil {
		return err
	}
	v.s = &parsed
	return nil
}

func (v *mapStringStringSliceFlag) Get() interface{} {
	return *v.s
}

func (v *mapStringStringSliceFlag) String() string {
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

type mapStringStringFlag struct {
	s *map[string]string
}

func (v *mapStringStringFlag) Set(s string) error {
	parsed, err := parsestring.ParseMap(s, reflect.TypeOf(map[string]string{}))
	if err != nil {
		return err
	}
	castParsed := parsed.Interface().(map[string]string)
	v.s = &castParsed
	return nil
}

func (v *mapStringStringFlag) Get() interface{} {
	return *v.s
}

func (v *mapStringStringFlag) String() string {
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
