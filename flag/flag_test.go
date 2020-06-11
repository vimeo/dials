package flag

import (
	"bytes"
	"context"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/tagformat/caseconversion"
)

func TestDirectBasic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	type Embed struct {
		Foo string `dialsflag:"foofoo"`
		Bar bool   // will have dials tag "Bar" after flatten mangler
	}
	type Config struct {
		Hello string
		World bool `dials:"world"`
		Embed
	}
	fs := flag.NewFlagSet("test flags", flag.ContinueOnError)
	src := &Set{
		Flags: fs,
		ParseFunc: func() error {
			return fs.Parse([]string{"-world", "-Hello=foobar", "-foofoo=something", "-Bar"})
		},
	}
	buf := &bytes.Buffer{}
	src.Flags.SetOutput(buf)

	d, err := dials.Config(ctx, &Config{}, src)
	if err != nil {
		t.Fatal(err)
	}
	src.Flags.Usage()
	t.Log(buf.String())

	got, ok := d.View().(*Config)
	if !ok {
		t.Fatalf("want: *Config, got: %T", got)
	}
	t.Logf("%+v", got)
	if got.Hello != "foobar" {
		t.Errorf("expected \"foobar\" for Hello, got %q", got.Hello)
	}
	if !got.World {
		t.Errorf("expected World to be true, got %t", got.World)
	}

	if got.Foo != "something" {
		t.Errorf("expected \"something\" for Foo, got %q", got.Foo)
	}

	if !got.Bar {
		t.Errorf("expected Bar to be true, got %t", got.Bar)
	}
}

type tu struct {
	Text string
}

// need a concrete type that implements TextUnmarshaler
func (u tu) UnmarshalText(data []byte) error {
	u.Text = string(data)
	return nil
}

func TestTable(t *testing.T) {
	for _, itbl := range []struct {
		name     string
		tmpl     interface{}
		args     []string
		expected interface{}
		expErr   string
	}{
		{
			name:     "basic_int_defaulted",
			tmpl:     &struct{ A int }{A: 4},
			args:     []string{},
			expected: &struct{ A int }{A: 4},
		},
		{
			name:     "basic_int_set",
			tmpl:     &struct{ A int }{A: 4},
			args:     []string{"--A=42"},
			expected: &struct{ A int }{A: 42},
		},
		{
			name:     "basic_string_defaulted",
			tmpl:     &struct{ A string }{A: "foobar"},
			args:     []string{},
			expected: &struct{ A string }{A: "foobar"},
		},
		{
			name:     "basic_string_set",
			tmpl:     &struct{ A string }{A: "foobar"},
			args:     []string{"--A=bizzleboodle"},
			expected: &struct{ A string }{A: "bizzleboodle"},
		},
		{
			name:     "basic_int16_default",
			tmpl:     &struct{ A int16 }{A: 10},
			args:     []string{},
			expected: &struct{ A int16 }{A: 10},
		},
		{
			name:     "basic_int16_set_nooverflow",
			tmpl:     &struct{ A int16 }{A: 10},
			args:     []string{"--A=128"},
			expected: &struct{ A int16 }{A: 128},
		},
		{
			name:     "basic_int16_set_overflow",
			tmpl:     &struct{ A int16 }{A: 10},
			args:     []string{"--A=1000000"},
			expected: nil,
			expErr:   "value for flag \"A\" (1000000) would overflow type int16",
		},
		{
			name:     "basic_int8_default",
			tmpl:     &struct{ A int8 }{A: 10},
			args:     []string{},
			expected: &struct{ A int8 }{A: 10},
		},
		{
			name:     "basic_int8_set_nooverflow",
			tmpl:     &struct{ A int8 }{A: 10},
			args:     []string{"--A=125"},
			expected: &struct{ A int8 }{A: 125},
		},
		{
			name:     "basic_int8_set_overflow",
			tmpl:     &struct{ A int8 }{A: 10},
			args:     []string{"--A=1000000"},
			expected: nil,
			expErr:   "value for flag \"A\" (1000000) would overflow type int8",
		},
		{
			name:     "map_string_string_set",
			tmpl:     &struct{ A map[string]string }{A: map[string]string{"z": "i"}},
			args:     []string{"--A=l:v"},
			expected: &struct{ A map[string]string }{A: map[string]string{"l": "v"}},
		},
		{
			name:     "map_string_string_default",
			tmpl:     &struct{ A map[string]string }{A: map[string]string{"z": "i"}},
			args:     []string{},
			expected: &struct{ A map[string]string }{A: map[string]string{"z": "i"}},
		},
		{
			name:     "map_string_string_slice_set",
			tmpl:     &struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}},
			args:     []string{"--A=l:v,l:z"},
			expected: &struct{ A map[string][]string }{A: map[string][]string{"l": {"v", "z"}}},
		},
		{
			name:     "map_string_string_slice_default",
			tmpl:     &struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}},
			args:     []string{},
			expected: &struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}},
		},
		{
			name:     "string_slice_set",
			tmpl:     &struct{ A []string }{A: []string{"i"}},
			args:     []string{"--A=v"},
			expected: &struct{ A []string }{A: []string{"v"}},
		},
		{
			name:     "string_slice_default",
			tmpl:     &struct{ A []string }{A: []string{"i"}},
			args:     []string{},
			expected: &struct{ A []string }{A: []string{"i"}},
		},
		{
			name:     "string_set_set",
			tmpl:     &struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}},
			args:     []string{"--A=v"},
			expected: &struct{ A map[string]struct{} }{A: map[string]struct{}{"v": {}}},
		},
		{
			name:     "string_set_default",
			tmpl:     &struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}},
			args:     []string{},
			expected: &struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}},
		},
		{
			name:     "basic_duration_default",
			tmpl:     &struct{ A time.Duration }{A: 10 * time.Nanosecond},
			args:     []string{},
			expected: &struct{ A time.Duration }{A: 10 * time.Nanosecond},
		},
		{
			name:     "basic_duration_set",
			tmpl:     &struct{ A time.Duration }{A: 10 * time.Nanosecond},
			args:     []string{"--A=3ms"},
			expected: &struct{ A time.Duration }{A: 3 * time.Millisecond},
		},
		{
			// use time.Time for a of couple test-cases since it implements TextUnmarshaler
			name:     "marshaler_time_set",
			tmpl:     &struct{ A time.Time }{A: time.Time{}},
			args:     []string{"--A=2019-12-18T14:00:12Z"},
			expected: &struct{ A time.Time }{A: time.Date(2019, time.December, 18, 14, 00, 12, 0, time.UTC)},
		},
		{
			name:     "marshaler_time_default",
			tmpl:     &struct{ A time.Time }{A: time.Time{}},
			args:     []string{},
			expected: &struct{ A time.Time }{A: time.Time{}},
		},
		{
			name:     "complex128_default",
			tmpl:     &struct{ A complex128 }{A: 10 + 3i},
			args:     []string{},
			expected: &struct{ A complex128 }{A: 10 + 3i},
		},
		{
			name:     "complex128_set_nooverflow",
			tmpl:     &struct{ A complex128 }{A: 10 + 3i},
			args:     []string{"--A=128+4i"},
			expected: &struct{ A complex128 }{A: 128 + 4i},
		},
		{
			name:     "complex64_default",
			tmpl:     &struct{ A complex64 }{A: 10 + 3i},
			args:     []string{},
			expected: &struct{ A complex64 }{A: 10 + 3i},
		},
		{
			name:     "complex64_set_nooverflow",
			tmpl:     &struct{ A complex64 }{A: 10 + 3i},
			args:     []string{"--A=128+4i"},
			expected: &struct{ A complex64 }{A: 128 + 4i},
		},
		{
			name:     "hierarchical_int_defaulted",
			tmpl:     &struct{ F struct{ A int } }{F: struct{ A int }{A: 4}},
			args:     []string{},
			expected: &struct{ F struct{ A int } }{F: struct{ A int }{A: 4}},
		},
		{
			name:     "hierarchical_int_set",
			tmpl:     &struct{ F struct{ A int } }{F: struct{ A int }{A: 4}},
			args:     []string{"--F-A=42"},
			expected: &struct{ F struct{ A int } }{F: struct{ A int }{A: 42}},
		},
		{
			name:     "hierarchical_ints_set",
			tmpl:     &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}},
			args:     []string{"--F-A=42", "--F-B=4123"},
			expected: &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 42, B: 4123}},
		},
		{
			name:     "hierarchical_ints_defaulted",
			tmpl:     &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}},
			args:     []string{},
			expected: &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}},
		},
		{
			name: "hierarchical_ints_multi_struct_set",
			tmpl: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 4, B: 34}, G: struct{ A int }{A: 5234}},
			args: []string{"--F-A=42", "--F-B=4123", "--G-A=5"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 4123}, G: struct{ A int }{A: 5}},
		},
		{
			name: "hierarchical_ints_multi_struct_partially_defaulted",
			tmpl: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 4, B: 34}, G: struct{ A int }{A: 5234}},
			args: []string{"--F-A=42", "--G-A=5"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 34}, G: struct{ A int }{A: 5}},
		},
		{
			name: "hierarchical_ints_multi_struct_partially_defaulted _with_tags",
			tmpl: &struct {
				F struct {
					A int `dials:"NotA"`
					B int
				}
				G struct {
					A int `dialsflag:"NotB"`
				}
			}{F: struct {
				A int `dials:"NotA"`
				B int
			}{
				A: 4, B: 34,
			},
				G: struct {
					A int `dialsflag:"NotB"`
				}{A: 5234}},
			args: []string{"--F-NotA=42", "--NotB=76"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 34}, G: struct{ A int }{A: 76}},
		}, {
			name: "non_pointer_text_unmarshal_implementation",
			tmpl: &struct {
				T tu
			}{T: tu{
				Text: "Hello",
			}},
			args: []string{"--T=foobar"},
			expected: &struct {
				T tu
			}{T: tu{
				Text: "Hello", //shouldn't change since it's non-pointer
			}},
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			// use UpperSnakeCase instead of default (CamelCase) since
			// single character field names like A and B make it hard to decode
			// between different fields
			nameConfig := &NameConfig{
				FieldNameEncodeCasing: caseconversion.EncodeUpperSnakeCase,
				TagEncodeCasing:       caseconversion.EncodeKebabCase,
			}
			s, setupErr := NewSetWithArgs(nameConfig, tbl.tmpl, tbl.args)
			require.NoError(t, setupErr, "failed to setup Set")

			d, cfgErr := dials.Config(ctx, tbl.tmpl, s)
			if tbl.expErr != "" {
				require.EqualError(t, cfgErr, tbl.expErr)
				return
			}
			require.NoError(t, cfgErr, "failed to stack/Value()")
			assert.EqualValues(t, tbl.expected, d.View())
		})
	}
}
