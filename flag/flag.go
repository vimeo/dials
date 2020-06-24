package flag

import (
	"encoding"
	"flag"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/flag/flaghelper"
	"github.com/vimeo/dials/ptrify"
	"github.com/vimeo/dials/tagformat/caseconversion"
	"github.com/vimeo/dials/transform"
)

var (
	timeDuration         = reflect.TypeOf(time.Nanosecond)
	flagReflectType      = reflect.TypeOf((*flag.Value)(nil)).Elem()
	stringSlice          = reflect.SliceOf(reflect.TypeOf(""))
	mapStringStringSlice = reflect.MapOf(reflect.TypeOf(""), stringSlice)
	mapStringString      = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(""))
	stringSet            = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(struct{}{}))
	textMReflectType     = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

	// Verify that Set implements the dials.Source interface
	_ dials.Source = (*Set)(nil)
)

const dialsFlagTag = "dialsflag"

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

// NewCmdLineSet registers flags for the passed template value in the standard
// library's main flag.CommandLine FlagSet so binaries using dials for flag
// configuration can play nicely with libraries that register flags with the
// standard library. (or libraries using dials can register flags and let the
// actual process's Main() call Parse())
func NewCmdLineSet(cfg *NameConfig, template interface{}) (*Set, error) {
	pval, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	s := Set{
		Flags:           flag.CommandLine,
		ParseFunc:       func() error { flag.Parse(); return nil },
		ptrType:         ptyp,
		flagsRegistered: true,
		NameCfg:         cfg,
		flagFieldName:   map[string]string{},
	}

	if err := s.registerFlags(pval, ptyp); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewSetWithArgs creates a new FlagSet and registers flags in it
func NewSetWithArgs(cfg *NameConfig, template interface{}, args []string) (*Set, error) {
	pval, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	fs := flag.NewFlagSet("", flag.ContinueOnError)

	s := Set{
		Flags:           fs,
		ParseFunc:       func() error { return fs.Parse(args) },
		ptrType:         ptyp,
		flagsRegistered: true,
		NameCfg:         cfg,
		flagFieldName:   map[string]string{},
	}

	if err := s.registerFlags(pval, ptyp); err != nil {
		return nil, err
	}

	return &s, nil
}

const (
	// HelpTextTag is the name of the struct tags for flag descriptions
	HelpTextTag = "dialsdesc"
	// DefaultFlagHelpText is the default help-text for fields with an
	// unset dialsdesc tag.
	DefaultFlagHelpText = "unset description (`" + HelpTextTag + "` struct tag)"
)

// Set is a flagset
type Set struct {
	Flags     *flag.FlagSet
	ParseFunc func() error

	ptrType reflect.Type

	// NameCfg defines tunables for constructing flag-names
	NameCfg *NameConfig

	flagsRegistered bool
	tfmr            *transform.Transformer
	trnslVal        reflect.Value
	// Map to store the flag name (key) and field name (value)
	flagFieldName map[string]string
}

func (s *Set) parse() error {
	if s.ParseFunc == nil {
		return fmt.Errorf("unparsed flagset with no ParseFunc set")
	}
	if err := s.ParseFunc(); err != nil {
		return fmt.Errorf("failed to parse flags: %s", err)
	}
	return nil
}

func (s *Set) registerFlags(tmpl reflect.Value, ptyp reflect.Type) error {
	fm := transform.NewFlattenMangler(transform.DialsTagName, s.NameCfg.FieldNameEncodeCasing, s.NameCfg.TagEncodeCasing)
	tfmr := transform.NewTransformer(ptyp, fm)
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
		if x, ok := sf.Tag.Lookup(HelpTextTag); ok {
			help = x
		}

		name := s.mkname(sf)
		s.flagFieldName[name] = sf.Name

		// if the flag already exists, don't register so the user can override
		// our behavior
		if s.Flags.Lookup(name) != nil {
			continue
		}

		// if the dialsflag tag has a hyphen (ex: `dialsflag:"-"`), don't
		// register the flag
		if dft, ok := sf.Tag.Lookup(dialsFlagTag); ok && (dft == "-") {
			continue
		}

		ft := sf.Type

		k := ft.Kind()
		for k == reflect.Ptr {
			ft = ft.Elem()
			k = ft.Kind()
		}
		isValue := ft.Implements(flagReflectType) || reflect.PtrTo(ft).Implements(flagReflectType)
		isTextM := ft.Implements(textMReflectType) || reflect.PtrTo(ft).Implements(textMReflectType)

		// get the concrete value of the field from the template
		fieldVal := transform.GetField(sf, tmpl)

		switch {
		case isValue:
			{

				newVal := fieldVal.Addr().Interface()
				s.Flags.Var(newVal.(flag.Value), name, help)
				continue
			}
		case isTextM:
			{
				// Make sure our newVal value actually points to something.
				newVal := fieldVal.Addr().Interface().(encoding.TextUnmarshaler)
				s.Flags.Var(flaghelper.NewMarshalWrapper(newVal), name, help)
				continue
			}
		case fieldVal.Type() == timeDuration:
			s.Flags.Duration(name, fieldVal.Interface().(time.Duration), help)
			continue
		default:
		}

		switch k {
		case reflect.String:
			s.Flags.String(name, fieldVal.Interface().(string), help)
		case reflect.Bool:
			s.Flags.Bool(name, fieldVal.Interface().(bool), help)
		case reflect.Float64:
			s.Flags.Float64(name, fieldVal.Interface().(float64), help)
		case reflect.Float32:
			s.Flags.Float64(name, float64(fieldVal.Interface().(float32)), help)
		case reflect.Complex64:
			s.Flags.Var(flaghelper.NewComplex64Var(fieldVal.Addr().Interface().(*complex64)), name, help)
		case reflect.Complex128:
			s.Flags.Var(flaghelper.NewComplex128Var(fieldVal.Addr().Interface().(*complex128)), name, help)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			{
				fvToInt := func() int {
					switch v := fieldVal.Interface().(type) {
					case int:
						return v
					case int8:
						return int(v)
					case int16:
						return int(v)
					case int32:
						return int(v)
					default:
						return 0
					}
				}
				s.Flags.Int(name, fvToInt(), help)
			}
		case reflect.Int64:
			s.Flags.Int64(name, fieldVal.Int(), help)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			{
				fvToInt := func() uint {
					switch v := fieldVal.Interface().(type) {
					case uint:
						return v
					case uint8:
						return uint(v)
					case uint16:
						return uint(v)
					case uint32:
						return uint(v)
					default:
						return 0
					}
				}
				s.Flags.Uint(name, fvToInt(), help)
			}
		case reflect.Uint64:
			s.Flags.Uint64(name, fieldVal.Interface().(uint64), help)
		case reflect.Slice, reflect.Map:
			switch ft {
			case stringSlice:
				s.Flags.Var(flaghelper.NewStringSliceFlag(fieldVal.Addr().Interface().(*[]string)), name, help)
			case mapStringStringSlice:
				s.Flags.Var(flaghelper.NewMapStringStringSliceFlag(fieldVal.Addr().Interface().(*map[string][]string)), name, help)
			case mapStringString:
				s.Flags.Var(flaghelper.NewMapStringStringFlag(fieldVal.Addr().Interface().(*map[string]string)), name, help)
			case stringSet:
				s.Flags.Var(flaghelper.NewStringSetFlag(fieldVal.Addr().Interface().(*map[string]struct{})), name, help)
			default:
				return fmt.Errorf("unhandled type %s", ft)
			}
		default:
			return fmt.Errorf("unhandled type %s", ft)
		}
	}
	return nil
}

// Value fills in the user-provided config struct using flags. It looks up the
// flags to bind into a given struct field by using that field's `dialsflag`
// struct tag if present, then its `dials` tag if present, and finally its name.
// If the struct has nested fields, Value will flatten the fields so flags can
// be defined for nested fields.
func (s *Set) Value(t *dials.Type) (reflect.Value, error) {
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
	if s.Flags == nil {
		// TODO: remove this fallback
		s.Flags = flag.NewFlagSet("", flag.ContinueOnError)
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
			return reflect.Value{}, fmt.Errorf("failed to parse: %s", err)
		}
	}
	var setErr error
	val := reflect.New(t.Type())
	s.Flags.Visit(func(f *flag.Flag) {
		fieldName, ok := s.flagFieldName[f.Name]
		if !ok {
			return
		}

		ffield := s.trnslVal.FieldByName(fieldName)
		if !ffield.IsNil() {
			// there's a 1:1 mapping between flags and field names so panic if
			// this happens
			panic(fmt.Errorf("Field name %s with flag %s is nil", fieldName, f.Name))
		}

		// We'll assume we're in a pointerified struct that matches
		// what we expected before, here.
		ptrVal := reflect.New(stripTypePtr(ffield.Type()))

		g, ok := f.Value.(flag.Getter)
		if !ok {
			return
		}

		fval := reflect.ValueOf(g.Get())
		switch fval.Type() {
		case ffield.Type().Elem():
			ptrVal.Elem().Set(fval)
			ffield.Set(ptrVal)
			return
		case ffield.Type():
			ffield.Set(fval)
			return
		}

		if willOverflow(fval, ptrVal.Elem()) {
			setErr = fmt.Errorf("value for flag %q (%s) would overflow type %s",
				f.Name, f.Value.String(), ptrVal.Type().Elem())
			return
		}
		cfval := fval.Convert(stripTypePtr(ffield.Type()))
		switch ffield.Kind() {
		case reflect.Ptr:
			// common case
			ptrVal.Elem().Set(cfval)
			ffield.Set(ptrVal)
		default:
			ffield.Set(cfval)
		}
		return
	})
	if setErr != nil {
		return val.Elem(), setErr
	}

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

func willOverflow(val, target reflect.Value) bool {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := val.Int()
		return target.OverflowInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := val.Uint()
		return target.OverflowUint(v)
	case reflect.Float32, reflect.Float64:
		v := val.Float()
		return target.OverflowFloat(v)
	case reflect.Complex64, reflect.Complex128:
		v := val.Complex()
		return target.OverflowComplex(v)
	default:
		return false
	}

}

// mkname creates a flag name based on the values of the dialsflag/dials tag or
// decoded field name and converting it into kebab case
func (s *Set) mkname(sf reflect.StructField) string {
	// use the name from the dialsflag tag for the flag name
	if name, ok := sf.Tag.Lookup(dialsFlagTag); ok {
		return name
	}
	// check if the dials tag is populated (it should be once it goes through
	// the flatten mangler).
	if name, ok := sf.Tag.Lookup(transform.DialsTagName); ok {
		return name
	}

	// panic because flatten mangler should set the dials tag so panic if that
	// wasn't set
	panic(fmt.Errorf("Expected dials tag name for struct field %q", sf.Name))

}
