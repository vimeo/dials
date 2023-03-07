package pflag

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/tagformat/caseconversion"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectBasicPFlag(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	type Embed struct {
		Foo      string `dialspflag:"foofoo"`
		Bar      bool   // will have dials tag "bar" after flatten mangler
		SomeTime time.Duration
	}
	type Config struct {
		Hello string
		World bool `dials:"world"`
		Embed
	}
	fs := pflag.NewFlagSet("test flags", pflag.ContinueOnError)
	fs.Usage = pflag.Usage
	src := &Set{
		Flags: fs,
		ParseFunc: func() error {
			return fs.Parse([]string{"--world", "--hello=foobar", "--foofoo=something", "--bar", "--some-time=2s"})
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

	got := d.View()

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

	if got.SomeTime != 2*time.Second {
		t.Errorf("expected SomeTime to be 2s, got %s", got.SomeTime)
	}
}

func TestDefaultVals(t *testing.T) {
	type otherString string
	type otherBool bool
	type otherInt int
	type otherInt8 int8
	type otherInt16 int16
	type otherInt32 int32
	type otherInt64 int64
	type otherUint uint
	type otherUint8 uint8
	type otherUint16 uint16
	type otherUint32 uint32
	type otherUint64 uint64
	type otherFloat32 float32
	type otherFloat64 float64
	type otherComplex64 complex64
	type otherComplex128 complex128

	type config struct {
		OString     otherString
		OBool       otherBool
		OInt        otherInt
		OInt8       otherInt8
		OInt16      otherInt16
		OInt32      otherInt32
		OInt64      otherInt64
		OUint       otherUint
		OUint8      otherUint8
		OUint16     otherUint16
		OUint32     otherUint32
		OUint64     otherUint64
		OFloat32    otherFloat32
		OFloat64    otherFloat64
		OComplex64  otherComplex64
		OComplex128 otherComplex128
	}

	c := config{
		OString:     "a-string",
		OBool:       true,
		OInt:        -1,
		OInt8:       -2,
		OInt16:      -3,
		OInt32:      -4,
		OInt64:      -5,
		OUint:       1,
		OUint8:      2,
		OUint16:     3,
		OUint32:     4,
		OUint64:     5,
		OFloat32:    6.0,
		OFloat64:    7.0,
		OComplex64:  8 + 2i,
		OComplex128: 9 + 3i,
	}

	expected := c

	fs := pflag.NewFlagSet("test flags", pflag.ContinueOnError)
	fs.Usage = pflag.Usage
	src := &Set{
		Flags: fs,
		ParseFunc: func() error {
			// don't need to parse any flags because we're only interested in
			// checking the default setting with these custom types.
			return fs.Parse([]string{})
		},
	}
	buf := &bytes.Buffer{}
	src.Flags.SetOutput(buf)

	d, err := dials.Config(context.Background(), &c, src)
	if err != nil {
		t.Fatal(err)
	}
	src.Flags.Usage()
	t.Log(buf.String())

	got := d.View()

	if *got != expected {
		t.Errorf("wanted %+v got %+v", expected, got)
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

func testWrapDials[T any](tmpl *T) func(ctx context.Context, src *Set) (any, error) {
	return func(ctx context.Context, src *Set) (any, error) {
		d, err := dials.Config(context.Background(), tmpl, src)
		if d == nil {
			return nil, err
		}
		return d.View(), err
	}
}

func TestPFlags(t *testing.T) {
	for _, itbl := range []struct {
		name string
		// returns the template and a callback for using the Set-typed source with dials
		tmplCB   func() (any, func(ctx context.Context, src *Set) (any, error))
		args     []string
		expected any
		expErr   string
	}{
		{
			name: "basic_int_defaulted",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int }{A: 4}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A int }{A: 4},
		},
		{
			name: "basic_int_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int }{A: 4}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=42"},
			expected: &struct{ A int }{A: 42},
		},
		{
			name: "basic_float32_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A float32 }{A: 4.5}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=42.5"},
			expected: &struct{ A float32 }{A: 42.5},
		},
		{
			name: "basic_string_defaulted",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A string }{A: "foobar"}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A string }{A: "foobar"},
		},
		{
			name: "basic_string_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A string }{A: "foobar"}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=bizzleboodle"},
			expected: &struct{ A string }{A: "bizzleboodle"},
		},
		{
			name: "shorthand_string_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					A string `dialspflagshort:"b"`
				}{A: "foobar"}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"-b=bizzleboodle"},
			expected: &struct{ A string }{A: "bizzleboodle"},
		},
		{
			name: "basic_int16_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int16 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A int16 }{A: 10},
		},
		{
			name: "basic_int16_set_nooverflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int16 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=128"},
			expected: &struct{ A int16 }{A: 128},
		},
		{
			name: "basic_int16_set_overflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int16 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=1000000"},
			expected: nil,
			expErr:   "failed to parse pflags: invalid argument \"1000000\" for \"--a\" flag: strconv.ParseInt: parsing \"1000000\": value out of range",
		},
		{
			name: "basic_int8_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int8 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A int8 }{A: 10},
		},
		{
			name: "basic_int8_set_nooverflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int8 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=125"},
			expected: &struct{ A int8 }{A: 125},
		},
		{
			name: "basic_int8_set_overflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A int8 }{A: 10}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=1000000"},
			expected: nil,
			expErr:   "failed to parse pflags: invalid argument \"1000000\" for \"--a\" flag: strconv.ParseInt: parsing \"1000000\": value out of range",
		},
		{
			name: "map_string_string_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string]string }{A: map[string]string{"z": "i"}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=l:v"},
			expected: &struct{ A map[string]string }{A: map[string]string{"l": "v"}},
		},
		{
			name: "map_string_string_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string]string }{A: map[string]string{"z": "i"}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A map[string]string }{A: map[string]string{"z": "i"}},
		},
		{
			name: "map_string_string_slice_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=l:v,l:z"},
			expected: &struct{ A map[string][]string }{A: map[string][]string{"l": {"v", "z"}}},
		},
		{
			name: "map_string_string_slice_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A map[string][]string }{A: map[string][]string{"z": {"i"}}},
		},
		{
			name: "string_set_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=v"},
			expected: &struct{ A map[string]struct{} }{A: map[string]struct{}{"v": {}}},
		},
		{
			name: "string_set_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A map[string]struct{} }{A: map[string]struct{}{"i": {}}},
		},
		{
			name: "complex128_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A complex128 }{A: 10 + 3i}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A complex128 }{A: 10 + 3i},
		},
		{
			name: "complex128_set_nooverflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A complex128 }{A: 10 + 3i}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=128+4i"},
			expected: &struct{ A complex128 }{A: 128 + 4i},
		},
		{
			name: "complex64_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A complex64 }{A: 10 + 3i}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A complex64 }{A: 10 + 3i},
		},
		{
			name: "complex64_set_nooverflow",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A complex64 }{A: 10 + 3i}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=128+4i"},
			expected: &struct{ A complex64 }{A: 128 + 4i},
		},
		{
			name: "string_slice_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A []string }{A: []string{"i"}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=v"},
			expected: &struct{ A []string }{A: []string{"v"}},
		},
		{
			name: "string_slice_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A []string }{A: []string{"i"}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A []string }{A: []string{"i"}},
		},
		{
			name: "basic_duration_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A time.Duration }{A: 10 * time.Nanosecond}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A time.Duration }{A: 10 * time.Nanosecond},
		},
		{
			name: "basic_duration_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A time.Duration }{A: 10 * time.Nanosecond}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=3ms"},
			expected: &struct{ A time.Duration }{A: 3 * time.Millisecond},
		},
		{
			// use time.Time for a of couple test-cases since it implements TextUnmarshaler
			name: "marshaler_time_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A time.Time }{A: time.Time{}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--a=2019-12-18T14:00:12Z"},
			expected: &struct{ A time.Time }{A: time.Date(2019, time.December, 18, 14, 00, 12, 0, time.UTC)},
		},
		{
			name: "marshaler_time_default",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ A time.Time }{A: time.Time{}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ A time.Time }{A: time.Time{}},
		},

		{
			name: "hierarchical_int_defaulted",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ F struct{ A int } }{F: struct{ A int }{A: 4}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ F struct{ A int } }{F: struct{ A int }{A: 4}},
		},
		{
			name: "hierarchical_int_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ F struct{ A int } }{F: struct{ A int }{A: 4}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--f-a=42"},
			expected: &struct{ F struct{ A int } }{F: struct{ A int }{A: 42}},
		},
		{
			name: "hierarchical_ints_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{"--f-a=42", "--f-b=4123"},
			expected: &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 42, B: 4123}},
		},
		{
			name: "hierarchical_ints_defaulted",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}}
				return &cfg, testWrapDials(&cfg)
			},
			args:     []string{},
			expected: &struct{ F struct{ A, B int } }{F: struct{ A, B int }{A: 4, B: 34}},
		},
		{
			name: "hierarchical_ints_multi_struct_set",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					F struct{ A, B int }
					G struct{ A int }
				}{F: struct{ A, B int }{A: 4, B: 34}, G: struct{ A int }{A: 5234}}
				return &cfg, testWrapDials(&cfg)
			},
			args: []string{"--f-a=42", "--f-b=4123", "--g-a=5"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 4123}, G: struct{ A int }{A: 5}},
		},
		{
			name: "hierarchical_ints_multi_struct_partially_defaulted",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					F struct{ A, B int }
					G struct{ A int }
				}{F: struct{ A, B int }{A: 4, B: 34}, G: struct{ A int }{A: 5234}}
				return &cfg, testWrapDials(&cfg)
			},
			args: []string{"--f-a=42", "--g-a=5"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 34}, G: struct{ A int }{A: 5}},
		},
		{
			name: "hierarchical_ints_multi_struct_with_hypen",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					F struct{ A, B int }
					G struct {
						A int `dials:"-"`
					}
				}{F: struct{ A, B int }{A: 4, B: 34}, G: struct {
					A int `dials:"-"`
				}{A: 5234}}
				return &cfg, testWrapDials(&cfg)
			},
			args:   []string{"--f-a=42", "--g-a=5"},
			expErr: "failed to parse pflags: unknown flag: --g-a",
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 34}, G: struct{ A int }{A: 5234}},
		},
		{
			name: "hierarchical_ints_multi_struct_partially_defaulted _with_tags",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					F struct {
						A int `dials:"NotA"`
						B int
					}
					G struct {
						A int `dialspflag:"NotB"`
					}
				}{F: struct {
					A int `dials:"NotA"`
					B int
				}{
					A: 4, B: 34,
				},
					G: struct {
						A int `dialspflag:"NotB"`
					}{A: 5234}}
				return &cfg, testWrapDials(&cfg)
			},
			args: []string{"--f-NotA=42", "--NotB=76"},
			expected: &struct {
				F struct{ A, B int }
				G struct{ A int }
			}{F: struct{ A, B int }{A: 42, B: 34}, G: struct{ A int }{A: 76}},
		}, {
			name: "non_pointer_text_unmarshal_implementation",
			tmplCB: func() (any, func(ctx context.Context, src *Set) (any, error)) {
				cfg := struct {
					T tu
				}{T: tu{
					Text: "Hello",
				}}
				return &cfg, testWrapDials(&cfg)
			},
			args: []string{"--t=foobar"},
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
			tmpl, run := tbl.tmplCB()
			s, setupErr := NewSetWithArgs(nameConfig, tmpl, tbl.args)
			require.NoError(t, setupErr, "failed to setup Set")

			c, cfgErr := run(ctx, s)
			if tbl.expErr != "" {
				require.EqualError(t, cfgErr, tbl.expErr)
				return
			}
			require.NoError(t, cfgErr, "failed to stack/Value()")
			assert.EqualValues(t, tbl.expected, c)
		})
	}
}

func TestMust(t *testing.T) {
	type Config struct {
		Hello string
		World bool `dials:"world"`
	}

	fs := Must(NewSetWithArgs(DefaultFlagNameConfig(), &Config{}, []string{"--world", "--hello=foobar"}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, err := dials.Config(ctx, &Config{}, fs)
	if err != nil {
		t.Fatal(err)
	}

	got := d.View()

	if got.Hello != "foobar" {
		t.Errorf("expected \"foobar\" for Hello, got %q", got.Hello)
	}
	if !got.World {
		t.Errorf("expected World to be true, got %t", got.World)
	}
}
