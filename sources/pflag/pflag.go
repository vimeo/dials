package pflag

import (
	"context"
	"encoding"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/ptrify"
	"github.com/vimeo/dials/sources/flag/flaghelper"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/transform"

	"github.com/spf13/pflag"
)

var (
	// the following types are unsupported by the pflag package but are supported
	// in dials pflag package. We check for these types so we can handle them appropriately
	pflagReflectType     = reflect.TypeOf((*pflag.Value)(nil)).Elem()
	textMReflectType     = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	timeDuration         = reflect.TypeOf(time.Nanosecond)
	stringSlice          = reflect.SliceOf(reflect.TypeOf(""))
	mapStringStringSlice = reflect.MapOf(reflect.TypeOf(""), stringSlice)
	mapStringString      = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(""))
	stringSet            = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(struct{}{}))

	stringType = reflect.TypeOf("")

	boolType = reflect.TypeOf(false)

	float32Type = reflect.TypeOf(float32(0))
	float64Type = reflect.TypeOf(float64(0))

	intType   = reflect.TypeOf(int(0))
	int8Type  = reflect.TypeOf(int8(0))
	int16Type = reflect.TypeOf(int16(0))
	int32Type = reflect.TypeOf(int32(0))
	int64Type = reflect.TypeOf(int64(0))

	uintType    = reflect.TypeOf(uint(0))
	uint8Type   = reflect.TypeOf(uint8(0))
	uint16Type  = reflect.TypeOf(uint16(0))
	uint32Type  = reflect.TypeOf(uint32(0))
	uint64Type  = reflect.TypeOf(uint64(0))
	uintptrType = reflect.TypeOf(uintptr(0))

	complex64Type  = reflect.TypeOf((*complex64)(nil))
	complex128Type = reflect.TypeOf((*complex128)(nil))

	intSliceType   = reflect.SliceOf(intType)
	int8SliceType  = reflect.SliceOf(int8Type)
	int16SliceType = reflect.SliceOf(int16Type)
	int32SliceType = reflect.SliceOf(int32Type)
	int64SliceType = reflect.SliceOf(int64Type)

	uintSliceType    = reflect.SliceOf(uintType)
	uint8SliceType   = reflect.SliceOf(uint8Type)
	uint16SliceType  = reflect.SliceOf(uint16Type)
	uint32SliceType  = reflect.SliceOf(uint32Type)
	uint64SliceType  = reflect.SliceOf(uint64Type)
	uintptrSliceType = reflect.SliceOf(uintptrType)

	// Verify that Set implements the dials.Source interface
	_ dials.Source = (*Set)(nil)
)

const (
	// DefaultFlagHelpText is the default help-text for fields with an
	// unset dialsdesc tag.
	DefaultFlagHelpText = "unset description (`" + common.DialsHelpTextTag + "` struct tag)"
)

// NameConfig defines the parameters for separating components of a flag-name
type NameConfig struct {
	// FieldNameEncodeCasing is for the field names used by the flatten mangler
	FieldNameEncodeCasing caseconversion.EncodeCasingFunc
	// TagEncodeCasing is for the tag names used by the flatten mangler
	TagEncodeCasing caseconversion.EncodeCasingFunc
}

// TODO(@sachi): update FieldNameEncodeCasing to EncodeGoCamelCase once it exists

// DefaultFlagNameConfig defines a reasonably-defaulted NameConfig for field names
// and tags
func DefaultFlagNameConfig() *NameConfig {
	return &NameConfig{
		FieldNameEncodeCasing: caseconversion.EncodeUpperCamelCase,
		TagEncodeCasing:       caseconversion.EncodeKebabCase,
	}
}

func ptrified(template interface{}) (reflect.Value, reflect.Type, error) {
	val := reflect.ValueOf(template)
	if val.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, fmt.Errorf("non-pointer-type passed: %s", val.Type())
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return reflect.Value{}, nil, fmt.Errorf("pointer-to-non-struct-type passed: %s", val.Type())
	}
	typ := val.Type()
	out := ptrify.Pointerify(typ, val)
	return val, out, nil
}

// NewCmdLineSet registers flags for the passed template value in the library's
// pflag.CommandLine FlagSet so binaries using dials for flag
// configuration can play nicely with libraries that register flags with the
// pflag library. (or libraries using dials can register flags and let the
// actual process's Main() call Parse())
func NewCmdLineSet(cfg *NameConfig, template interface{}) (*Set, error) {
	fs := pflag.CommandLine
	parseFunc := func() error { pflag.Parse(); return nil }

	return newSet(cfg, template, fs, parseFunc)
}

// NewSetWithArgs creates a new pflag FlagSet and registers flags in it
func NewSetWithArgs(cfg *NameConfig, template interface{}, args []string) (*Set, error) {

	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	parseFunc := func() error { return fs.Parse(args) }

	return newSet(cfg, template, fs, parseFunc)
}

// NewSetWithFlagSet uses the passed in pflag FlagSet and registers flags
func NewSetWithFlagSet(cfg *NameConfig, template interface{}, flagset *pflag.FlagSet) (*Set, error) {
	return newSet(cfg, template, flagset, nil)
}

// NewDefaultSetWithFlagSet uses the passedin pflag FlagSet and registers flags
// with the DefaultFlagNameConfig
func NewDefaultSetWithFlagSet(template interface{}, flagset *pflag.FlagSet) (*Set, error) {
	return newSet(DefaultFlagNameConfig(), template, flagset, nil)
}

// Must is a helper that wraps a call to a function returning (*Set, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//
//	var flagset = pflag.Must(pflag.NewCmdLineSet(pflag.DefaultFlagNameConfig(), config))
func Must(s *Set, err error) *Set {
	if err != nil {
		panic(fmt.Errorf("error registering flags: %w", err))
	}
	return s
}

// newSet is a helper function to initialize Set and register flags
func newSet(cfg *NameConfig, template interface{}, fs *pflag.FlagSet, parseFunc func() error) (*Set, error) {
	pval, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	s := Set{
		Flags:           fs,
		ParseFunc:       parseFunc,
		ptrType:         ptyp,
		flagsRegistered: true,
		NameCfg:         cfg,
		flagFieldName:   map[string]string{},
		flagValues:      map[string]reflect.Value{},
	}

	if err := s.registerFlags(pval, ptyp); err != nil {
		return nil, err
	}

	return &s, nil

}

// Set source is provided for compatibility with the cobra command line
// framework. Others should prefer to use flag.Set
type Set struct {
	Flags     *pflag.FlagSet
	ParseFunc func() error

	ptrType reflect.Type

	// NameCfg defines tunables for constructing flag-names
	NameCfg *NameConfig

	flagsRegistered bool
	tfmr            *transform.Transformer
	trnslVal        reflect.Value
	// Map to store the flag name (key) and field name (value)
	flagFieldName map[string]string
	flagValues    map[string]reflect.Value
}

func (s *Set) parse() error {
	// ParseFunc will be nil when ge
	if s.ParseFunc == nil {
		return nil
	}
	if err := s.ParseFunc(); err != nil {
		return fmt.Errorf("failed to parse pflags: %w", err)
	}
	return nil
}

func (s *Set) registerFlags(tmpl reflect.Value, ptyp reflect.Type) error {
	fm := transform.NewFlattenMangler(common.DialsTagName, s.NameCfg.FieldNameEncodeCasing, s.NameCfg.TagEncodeCasing)
	tfmr := transform.NewTransformer(ptyp, transform.NewAliasMangler(common.DialsTagName, common.DialsPFlagTag, common.DialsPFlagShortTag), fm)
	val, TrnslErr := tfmr.Translate()
	if TrnslErr != nil {
		return TrnslErr
	}

	s.tfmr = tfmr
	s.trnslVal = val

	t := val.Type()

	k := t.Kind()
	for k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}

	// the input kind will be struct after calling Translate on it
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		help := DefaultFlagHelpText
		if x, ok := sf.Tag.Lookup(common.DialsHelpTextTag); ok {
			help = x
		}

		name := s.mkname(sf)
		s.flagFieldName[name] = sf.Name

		// if the flag already exists, don't register so the user can override
		// our behavior
		if s.Flags.Lookup(name) != nil {
			continue
		}

		// If the field's dialspflag tag is a hyphen (ex: `dialspflag:"-"`),
		// don't register the flag. Currently nested fields with "-" tag will
		// still be registered
		if dpt, ok := sf.Tag.Lookup(common.DialsPFlagTag); ok && (dpt == "-") {
			continue
		}

		ft := sf.Type

		k := ft.Kind()
		for k == reflect.Ptr {
			ft = ft.Elem()
			k = ft.Kind()
		}
		isValue := ft.Implements(pflagReflectType) || reflect.PtrTo(ft).Implements(pflagReflectType)
		isTextM := ft.Implements(textMReflectType) || reflect.PtrTo(ft).Implements(textMReflectType)

		// get the concrete value of the field from the template
		fieldVal := transform.GetField(sf, tmpl)
		shorthand, _ := sf.Tag.Lookup(common.DialsPFlagShortTag)
		var f interface{}

		switch {
		case isValue:
			{
				s.Flags.VarP(fieldVal.Addr().Interface().(pflag.Value), name, shorthand, help)
				s.flagValues[name] = fieldVal.Addr()
				continue
			}
		case isTextM:
			{
				// Make sure our newVal value actually points to something.
				newVal := fieldVal.Addr().Interface().(encoding.TextUnmarshaler)
				s.Flags.VarP(flaghelper.NewMarshalWrapper(newVal), name, shorthand, help)
				s.flagValues[name] = fieldVal.Addr()
				continue
			}
		case fieldVal.Type() == timeDuration:
			f = s.Flags.DurationP(name, shorthand, fieldVal.Interface().(time.Duration), help)
			s.flagValues[name] = reflect.ValueOf(f)
			continue
		default:
		}

		switch k {
		case reflect.String:
			f = s.Flags.StringP(name, shorthand, fieldVal.Convert(stringType).Interface().(string), help)
		case reflect.Bool:
			f = s.Flags.BoolP(name, shorthand, fieldVal.Convert(boolType).Interface().(bool), help)
		case reflect.Float64:
			f = s.Flags.Float64P(name, shorthand, fieldVal.Convert(float64Type).Interface().(float64), help)
		case reflect.Float32:
			f = s.Flags.Float32P(name, shorthand, fieldVal.Convert(float32Type).Interface().(float32), help)
		case reflect.Complex64:
			f = fieldVal.Addr().Interface()
			s.Flags.VarP(flaghelper.NewComplex64Var(fieldVal.Addr().Convert(complex64Type).Interface().(*complex64)), name, shorthand, help)
		case reflect.Complex128:
			f = fieldVal.Addr().Interface()
			s.Flags.VarP(flaghelper.NewComplex128Var(fieldVal.Addr().Convert(complex128Type).Interface().(*complex128)), name, shorthand, help)
		case reflect.Int:
			f = s.Flags.IntP(name, shorthand, fieldVal.Convert(intType).Interface().(int), help)
		case reflect.Int8:
			f = s.Flags.Int8P(name, shorthand, fieldVal.Convert(int8Type).Interface().(int8), help)
		case reflect.Int16:
			f = s.Flags.Int16P(name, shorthand, fieldVal.Convert(int16Type).Interface().(int16), help)
		case reflect.Int32:
			f = s.Flags.Int32P(name, shorthand, fieldVal.Convert(int32Type).Interface().(int32), help)
		case reflect.Int64:
			f = s.Flags.Int64P(name, shorthand, fieldVal.Convert(int64Type).Int(), help)
		case reflect.Uint:
			f = s.Flags.UintP(name, shorthand, fieldVal.Convert(uintType).Interface().(uint), help)
		case reflect.Uint8:
			f = s.Flags.Uint8P(name, shorthand, fieldVal.Convert(uint8Type).Interface().(uint8), help)
		case reflect.Uint16:
			f = s.Flags.Uint16P(name, shorthand, fieldVal.Convert(uint16Type).Interface().(uint16), help)
		case reflect.Uint32:
			f = s.Flags.Uint32P(name, shorthand, fieldVal.Convert(uint32Type).Interface().(uint32), help)
		case reflect.Uint64:
			f = s.Flags.Uint64P(name, shorthand, fieldVal.Convert(uint64Type).Interface().(uint64), help)
		case reflect.Uintptr:
			f = s.Flags.Uint64P(name, shorthand, uint64(fieldVal.Convert(uintptrType).Interface().(uintptr)), help)
		case reflect.Slice, reflect.Map:
			switch ft {
			case stringSlice:
				f = s.Flags.StringSliceP(name, shorthand, fieldVal.Interface().([]string), help)
			case mapStringStringSlice:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewMapStringStringSliceFlag(fieldVal.Addr().Interface().(*map[string][]string)), name, shorthand, help)
			case mapStringString:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewMapStringStringFlag(fieldVal.Addr().Interface().(*map[string]string)), name, shorthand, help)
			case stringSet:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewStringSetFlag(fieldVal.Addr().Interface().(*map[string]struct{})), name, shorthand, help)

				// signed integral slices
			case intSliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewSignedIntegralSlice(f.(*[]int)), name, shorthand, help)
			case int8SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewSignedIntegralSlice(f.(*[]int8)), name, shorthand, help)
			case int16SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewSignedIntegralSlice(f.(*[]int16)), name, shorthand, help)
			case int32SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewSignedIntegralSlice(f.(*[]int32)), name, shorthand, help)
			case int64SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewSignedIntegralSlice(f.(*[]int64)), name, shorthand, help)

				// unsigned integral slices
			case uintSliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uint)), name, shorthand, help)
			case uint8SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uint8)), name, shorthand, help)
			case uint16SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uint16)), name, shorthand, help)
			case uint32SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uint32)), name, shorthand, help)
			case uint64SliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uint64)), name, shorthand, help)
			case uintptrSliceType:
				f = fieldVal.Addr().Interface()
				s.Flags.VarP(flaghelper.NewUnsignedIntegralSlice(f.(*[]uintptr)), name, shorthand, help)

			default:
				// Unhandled type. Just keep going.
				continue
			}

		default:
			// Unhandled type. Just keep going.
			continue
		}

		v := reflect.ValueOf(f)
		s.flagValues[name] = v
	}
	return nil
}

// Value fills in the user-provided config struct using flags. It looks up the
// flags to bind into a given struct field by using that field's `dialspflag`
// struct tag if present, then its `dials` tag if present, and finally its name.
// If the struct has nested fields, Value will flatten the fields so flags can
// be defined for nested fields.
func (s *Set) Value(_ context.Context, t *dials.Type) (reflect.Value, error) {
	// Check whether we've gone through the exercise of parsing flags yet
	// (and types are compatible).
	if s.ptrType != nil {
		if !t.Type().ConvertibleTo(s.ptrType) {
			return reflect.Value{}, fmt.Errorf(
				"incompatible types called with Value() (%s) and constructor for flag Source (%s)",
				t.Type(), s.ptrType)
		}
	}

	if s.flagFieldName == nil {
		s.flagFieldName = map[string]string{}
	}

	if s.flagValues == nil {
		s.flagValues = map[string]reflect.Value{}
	}
	if s.Flags == nil {
		// TODO: remove this fallback
		s.Flags = pflag.NewFlagSet("", pflag.ContinueOnError)
		if s.ParseFunc == nil {
			s.ParseFunc = func() error { return s.Flags.Parse(os.Args[1:]) }
		}
	}

	if s.NameCfg == nil {
		s.NameCfg = DefaultFlagNameConfig()
	}

	if !s.flagsRegistered {
		var ptyp reflect.Type
		if s.ptrType == nil {
			ptyp = t.Type()
		} else {
			ptyp = s.ptrType
		}

		if err := s.registerFlags(reflect.Value{}, ptyp); err != nil {
			return reflect.Value{}, err
		}
		s.flagsRegistered = true
	}
	if !s.Flags.Parsed() {
		if err := s.parse(); err != nil {
			return reflect.Value{}, err
		}
	}

	s.Flags.Visit(func(f *pflag.Flag) {
		fieldName, ok := s.flagFieldName[f.Name]
		if !ok {
			return
		}

		ffield := s.trnslVal.FieldByName(fieldName)
		if !ffield.IsNil() {
			// there's a 1:1 mapping between flags and field names so panic if
			// this happens
			panic(fmt.Errorf("field name %s with flag %s is nil", fieldName, f.Name))
		}

		// We'll assume we're in a pointerified struct that matches
		// what we expected before, here.
		ptrVal := reflect.New(stripTypePtr(ffield.Type()))
		fval, ok := s.flagValues[f.Name]
		if !ok {
			return
		}

		switch fval.Type() {
		case ffield.Type().Elem():
			ptrVal.Elem().Set(fval)
			ffield.Set(ptrVal)
			return
		case ffield.Type():
			ffield.Set(fval)
			return
		case ffield.Addr().Type(): // flag is a pointer (*[]string) and ffield isn't ([]string)
			ffield.Set(fval.Elem())
			return
		}

		// fval is always a pointer, so dereference it before converting to the final type
		cfval := fval.Elem().Convert(stripTypePtr(ffield.Type()))
		switch ffield.Kind() {
		case reflect.Ptr:
			// common case
			ptrVal.Elem().Set(cfval)
			ffield.Set(ptrVal)
		default:
			ffield.Set(cfval)
		}
	})

	return s.tfmr.ReverseTranslate(s.trnslVal)
}

func stripTypePtr(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr:
		return t.Elem()
	default:
		return t
	}
}

// mkname creates a flag name based on the values of the dialspflag/dials tag or
// decoded field name and converting it into kebab case
func (s *Set) mkname(sf reflect.StructField) string {
	// use the name from the dialspflag tag for the flag name
	if name, ok := sf.Tag.Lookup(common.DialsPFlagTag); ok {
		return name
	}
	// check if the dials tag is populated (it should be once it goes through
	// the flatten mangler).
	if name, ok := sf.Tag.Lookup(common.DialsTagName); ok {
		return name
	}

	// panic because flatten mangler should set the dials tag so panic if that
	// wasn't set
	panic(fmt.Errorf("expected dials tag name for struct field %q", sf.Name))

}
