package flag

import (
	"encoding"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/vimeo/dials"
	"github.com/vimeo/dials/transform"
)

var (
	// Verify that PflagSet implements the dials.Source interface
	_ dials.Source = (*PflagSet)(nil)
)

const (
	dialsPFlagTag      = "dialspflag"
	dialsPFlagShortTag = "dialspflagshort"
)

// NewCmdLinePflagSet registers flags for the passed template value in the standard
// library's main flag.CommandLine FlagSet so binaries using dials for flag
// configuration can play nicely with libraries that register flags with the
// standard library. (or libraries using dials can register flags and let the
// actual process's Main() call Parse())
func NewCmdLinePflagSet(cfg *NameConfig, template interface{}) (*PflagSet, error) {
	pval, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	s := PflagSet{
		Flags:           pflag.CommandLine,
		ParseFunc:       func() error { flag.Parse(); return nil },
		ptrType:         ptyp,
		flagsRegistered: true,
		NameCfg:         cfg,
		flagFieldName:   map[string]string{},
		flagValues:      map[string]interface{}{},
	}

	if err := s.registerPFlags(pval, ptyp); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewPFlagSetWithArgs creates a new FlagSet and registers flags in it
func NewPFlagSetWithArgs(cfg *NameConfig, template interface{}, args []string) (*PflagSet, error) {
	pval, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	fs := pflag.NewFlagSet("", pflag.ContinueOnError)

	s := PflagSet{
		Flags:           fs,
		ParseFunc:       func() error { return fs.Parse(args) },
		ptrType:         ptyp,
		flagsRegistered: true,
		NameCfg:         cfg,
		flagFieldName:   map[string]string{},
		flagValues:      map[string]interface{}{},
	}

	if err := s.registerPFlags(pval, ptyp); err != nil {
		return nil, err
	}

	return &s, nil
}

// PflagSet is a flagset. Please only use PflagSet when using cobra command line
// tool. Use Set for flags functionality for other cases
type PflagSet struct {
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
	flagValues    map[string]interface{}
}

func (s *PflagSet) parse() error {
	if s.ParseFunc == nil {
		return fmt.Errorf("unparsed flagset with no ParseFunc set")
	}
	if err := s.ParseFunc(); err != nil {
		return fmt.Errorf("failed to parse pflags: %s", err)
	}
	return nil
}

func (s *PflagSet) registerPFlags(tmpl reflect.Value, ptyp reflect.Type) error {
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
		shorthand, _ := sf.Tag.Lookup(dialsPFlagShortTag)
		var f interface{}

		switch {
		case isValue:
			{
				f = s.Flags.VarPF(fieldVal.Addr().Interface().(pflag.Value), name, shorthand, help)
				s.flagValues[name] = f
				continue
			}
		case isTextM:
			{
				// Make sure our newVal value actually points to something.
				newVal := fieldVal.Addr().Interface()
				f = s.Flags.VarPF(marshalWrapper{v: newVal.(encoding.TextUnmarshaler)}, name, shorthand, help)
				s.flagValues[name] = f
				continue
			}
		case fieldVal.Type() == timeDuration:
			f = s.Flags.DurationP(name, shorthand, fieldVal.Interface().(time.Duration), help)
			s.flagValues[name] = f
			continue
		default:
		}

		switch k {
		case reflect.String:
			f = s.Flags.StringP(name, shorthand, fieldVal.Interface().(string), help)
		case reflect.Bool:
			f = s.Flags.BoolP(name, shorthand, fieldVal.Interface().(bool), help)
		case reflect.Float64:
			f = s.Flags.Float64P(name, shorthand, fieldVal.Interface().(float64), help)
		case reflect.Float32:
			f = s.Flags.Float64P(name, shorthand, float64(fieldVal.Interface().(float32)), help)
		case reflect.Complex64:
			f = s.Flags.VarPF(&complex64Var{c: fieldVal.Interface().(complex64)}, name, shorthand, help)
		case reflect.Complex128:
			f = s.Flags.VarPF(&complex128Var{c: fieldVal.Interface().(complex128)}, name, shorthand, help)
		case reflect.Int:
			f = s.Flags.IntP(name, shorthand, fieldVal.Interface().(int), help)
		case reflect.Int8:
			f = s.Flags.Int8P(name, shorthand, fieldVal.Interface().(int8), help)
		case reflect.Int16:
			f = s.Flags.Int16P(name, shorthand, fieldVal.Interface().(int16), help)
		case reflect.Int32:
			f = s.Flags.Int32P(name, shorthand, fieldVal.Interface().(int32), help)
		case reflect.Int64:
			f = s.Flags.Int64P(name, shorthand, fieldVal.Int(), help)
		case reflect.Uint:
			f = s.Flags.UintP(name, shorthand, fieldVal.Interface().(uint), help)
		case reflect.Uint8:
			f = s.Flags.Uint8P(name, shorthand, fieldVal.Interface().(uint8), help)
		case reflect.Uint16:
			f = s.Flags.Uint16P(name, shorthand, fieldVal.Interface().(uint16), help)
		case reflect.Uint32:
			f = s.Flags.Uint32P(name, shorthand, fieldVal.Interface().(uint32), help)
		case reflect.Uint64:
			f = s.Flags.Uint64P(name, shorthand, fieldVal.Interface().(uint64), help)
		case reflect.Slice, reflect.Map:
			switch ft {
			case stringSlice:
				f = s.Flags.StringSliceP(name, shorthand, fieldVal.Interface().([]string), help)
			case mapStringStringSlice:
				f = s.Flags.VarPF(&mapStringStringSliceFlag{fieldVal.Addr().Interface().(*map[string][]string)}, name, shorthand, help)
			case mapStringString:
				f = s.Flags.VarPF(&mapStringStringFlag{fieldVal.Addr().Interface().(*map[string]string)}, name, shorthand, help)
			case stringSet:
				f = s.Flags.VarPF(&stringSetFlag{fieldVal.Addr().Interface().(*map[string]struct{})}, name, shorthand, help)
			default:
				return fmt.Errorf("unhandled type %s", ft)
			}

		default:
			return fmt.Errorf("unhandled type %s", ft)
		}

		s.flagValues[name] = f
	}
	return nil
}

// Value fills in the user-provided config struct using flags. It looks up the
// flags to bind into a given struct field by using that field's `dialspflag`
// struct tag if present, then its `dials` tag if present, and finally its name.
// If the struct has nested fields, Value will flatten the fields so flags can
// be defined for nested fields.
func (s *PflagSet) Value(t *dials.Type) (reflect.Value, error) {
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
		s.flagValues = map[string]interface{}{}
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

		if err := s.registerPFlags(reflect.Value{}, ptyp); err != nil {
			return reflect.Value{}, err
		}
		s.flagsRegistered = true
	}
	if !s.Flags.Parsed() {
		if err := s.parse(); err != nil {
			return reflect.Value{}, err
		}
	}
	var setErr error
	val := reflect.New(t.Type())
	s.Flags.Visit(func(f *pflag.Flag) {
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
		if g, ok := s.flagValues[f.Name]; ok {
			if flagGetter, isFlag := f.Value.(flag.Getter); isFlag {
				g = flagGetter.Get()
			}
			fval := reflect.ValueOf(g)

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
		}
	})
	if setErr != nil {
		return val.Elem(), setErr
	}

	return s.tfmr.ReverseTranslate(s.trnslVal)
}

// mkname creates a flag name based on the values of the dialspflag/dials tag or
// decoded field name and converting it into kebab case
func (s *PflagSet) mkname(sf reflect.StructField) string {
	// use the name from the dialspflag tag for the flag name
	if name, ok := sf.Tag.Lookup(dialsPFlagTag); ok {
		return name
	}
	// check if the dials tag is populated (it should be once it goes through
	// the flatten mangler).
	if name, ok := sf.Tag.Lookup(transform.DialsTagName); ok {
		return strings.ToLower(name)
	}

	// panic because flatten mangler should set the dials tag so panic if that
	// wasn't set
	panic(fmt.Errorf("Expected dials tag name for struct field %q", sf.Name))

}
