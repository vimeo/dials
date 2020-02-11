package flag

import (
	"encoding"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/ptrify"
)

var (
	timeDuration         = reflect.TypeOf(time.Nanosecond)
	flagValue            = reflect.TypeOf((*flag.Value)(nil)).Elem()
	stringSlice          = reflect.SliceOf(reflect.TypeOf(""))
	mapStringStringSlice = reflect.MapOf(reflect.TypeOf(""), stringSlice)
	mapStringString      = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(""))
	stringSet            = reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(struct{}{}))
	textMValue           = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

	// Verify that Set implements the dials.Source interface
	_ dials.Source = (*Set)(nil)
)

// NameConfig defines the parameters for separating components of a flag-name
type NameConfig struct {
	FieldSep string
	WordSep  string
}

// DashesNameConfig defines a reasonably-defaulted NameConfig with dashes for
// both separators.
func DashesNameConfig() NameConfig {
	return NameConfig{FieldSep: "-", WordSep: "-"}
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
func NewCmdLineSet(cfg NameConfig, template interface{}) (*Set, error) {
	tmpl, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	s := Set{
		Flags:           flag.CommandLine,
		ParseFunc:       func() error { flag.Parse(); return nil },
		ptrType:         ptyp,
		fieldPaths:      map[string][]int{},
		flagsRegistered: true,
		NameCfg:         cfg,
	}

	if err := s.walk("", []int{}, tmpl, ptyp); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewSetWithArgs creates a new FlagSet and registers flags in it
func NewSetWithArgs(cfg NameConfig, template interface{}, args []string) (*Set, error) {
	tmpl, ptyp, ptrifyErr := ptrified(template)
	if ptrifyErr != nil {
		return nil, ptrifyErr
	}

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	s := Set{
		Flags:           fs,
		ParseFunc:       func() error { return fs.Parse(args) },
		ptrType:         ptyp,
		fieldPaths:      map[string][]int{},
		flagsRegistered: true,
		NameCfg:         cfg,
	}

	if err := s.walk("", []int{}, tmpl, ptyp); err != nil {
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
	NameCfg NameConfig

	// Map from flag-name to field-path (offsets)
	fieldPaths map[string][]int

	flagsRegistered bool
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

func (s *Set) registerFieldFlag(name string, idxs []int, fieldVal reflect.Value, sf reflect.StructField) error {
	ft := sf.Type
	help := DefaultFlagHelpText
	if x, ok := sf.Tag.Lookup(HelpTextTag); ok {
		help = x
	}
	newValPtr := reflect.New(ft)
	ptr := newValPtr.Interface()
	k := fieldVal.Kind()

	isValue := ft.Implements(flagValue)
	isTextM := ft.Implements(textMValue)
	if k == reflect.Struct && !(isValue || isTextM) {
		if walkErr := s.walk(name+s.NameCfg.FieldSep, idxs, fieldVal, ft); walkErr != nil {
			return fmt.Errorf("failed to walk field %q: %s", name, walkErr)
		}
		return nil
	}

	switch {
	case isValue:
		{
			newVal := newValPtr.Elem().Interface()
			s.Flags.Var(newVal.(flag.Value), name, help)
			return nil
		}
	case isTextM:
		{
			// Make sure our newVal value actually points to something.
			newValPtr.Elem().Set(reflect.New(ft.Elem()))
			newVal := newValPtr.Elem().Interface()
			s.Flags.Var(marshalWrapper{v: newVal.(encoding.TextUnmarshaler)}, name, help)
			return nil
		}
	case fieldVal.Type() == timeDuration:
		s.Flags.Duration(name, fieldVal.Interface().(time.Duration), help)
		return nil
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
		s.Flags.Var(&complex64Var{c: fieldVal.Interface().(complex64)}, name, help)
	case reflect.Complex128:
		s.Flags.Var(&complex128Var{c: fieldVal.Interface().(complex128)}, name, help)
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
			s.Flags.Var(&stringSliceFlag{ptr.(*[]string)}, name, help)
		case mapStringStringSlice:
			s.Flags.Var(&mapStringStringSliceFlag{ptr.(*map[string][]string)}, name, help)
		case mapStringString:
			s.Flags.Var(&mapStringStringFlag{ptr.(*map[string]string)}, name, help)
		case stringSet:
			s.Flags.Var(&stringSetFlag{ptr.(*map[string]struct{})}, name, help)
		default:
			return fmt.Errorf("unhandled type %s",
				ft)
		}
	default:
		return fmt.Errorf("unhandled type %s", ft)
	}
	return nil
}

func stripPtrs(val reflect.Value) reflect.Value {
	for val.IsValid() {
		switch val.Kind() {
		case reflect.Ptr, reflect.Interface:
			val = val.Elem()
		default:
			return val
		}
	}
	return val
}

func (s *Set) walk(prefix string, pathIdxs []int, tmplVal reflect.Value, t reflect.Type) error {
	switch outerK := t.Kind(); outerK {
	case reflect.Struct:
	case reflect.Ptr:
		if t.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("flag: type passed was a pointer to a non-struct (%s)", t)
		}
		// Make sure to handle the pointerified field-values
		t = t.Elem()
	default:
		return fmt.Errorf("flag: type passed was not a struct or pointer to struct (%s)", t)
	}
	for i := 0; i < t.NumField(); i++ {
		// assemble the fieldname->index-list value
		idxs := make([]int, len(pathIdxs), len(pathIdxs)+1)
		copy(idxs, pathIdxs)
		idxs = append(idxs, i)

		sf := t.Field(i)
		// Make sure we're pointerized (or nilable)
		switch sf.Type.Kind() {
		case reflect.Ptr, reflect.Map, reflect.Slice:
		default:
			return fmt.Errorf("flag: programmer error: expected pointerized fields, got %s in %s",
				sf.Type, t)
		}

		fieldVal := reflect.Value{}
		if tmplVal.IsValid() {
			fieldVal = tmplVal.FieldByName(sf.Name)
		}
		fieldVal = stripPtrs(fieldVal)

		ft := sf.Type
		k := ft.Kind()
		for k == reflect.Ptr {
			ft = ft.Elem()
			k = ft.Kind()
		}

		if !fieldVal.IsValid() {
			fieldVal = reflect.Zero(ft)
		}
		name := prefix + mkname(t, sf)
		s.fieldPaths[name] = idxs
		// Do a lookup so that a caller can override our behavior.
		if s.Flags.Lookup(name) != nil {
			continue
		}
		if regErr := s.registerFieldFlag(name, idxs, fieldVal, sf); regErr != nil {
			return fmt.Errorf("failed to register flag(s) for field index %d in type %s: %s", i, t, regErr)
		}
	}
	return nil
}

// Value implements dials.Source, taking a dials type and returning a
// reflect.Value representing the values of the flags.
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

	if s.fieldPaths == nil {
		s.fieldPaths = map[string][]int{}
	}
	if s.Flags == nil {
		// TODO: remove this fallback
		s.Flags = flag.NewFlagSet("", flag.ContinueOnError)
		if s.ParseFunc == nil {
			s.ParseFunc = func() error { return s.Flags.Parse(os.Args[1:]) }
		}
	}
	if !s.flagsRegistered {
		if err := s.walk("", []int{}, reflect.Value{}, t.Type()); err != nil {
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
		fieldIdxs, ok := s.fieldPaths[f.Name]
		if !ok {
			return
		}
		// Iteratively initialize unset values
		for i := 1; i < len(fieldIdxs); i++ {
			interimField := val.Elem().FieldByIndex(fieldIdxs[:i])
			if !interimField.IsNil() {
				// If this field isn't nil, continue, we don't
				// want to overwrite a populated field with an
				// empty one. (since that would prevent us from
				// seeing other fields in the same (or
				// parallel) sub-structs.
				continue
			}
			interimPtr := reflect.New(interimField.Type().Elem())
			interimField.Set(interimPtr)
		}
		ffield := val.Elem().FieldByIndex(fieldIdxs)
		// We'll assume we're in a pointerified struct that matches
		// what we expected before, here.
		ptrVal := reflect.New(stripTypePtr(ffield.Type()))
		if g, ok := f.Value.(flag.Getter); ok {
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
		}
	})
	if setErr != nil {
		return val.Elem(), setErr
	}

	return val.Elem(), nil
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

// Mkname runs a heuristic to turn a struct path into a flag name.
//
// It doesn't handle intercapped acronyms.
func mkname(root reflect.Type, sf reflect.StructField) string {
	if name, ok := sf.Tag.Lookup("dialsflag"); ok {
		return name
	}
	if name, ok := sf.Tag.Lookup("dials"); ok {
		return name
	}
	b := &strings.Builder{}
	key := make([]int, len(sf.Index))
	for i, idx := range sf.Index {
		key[i] = idx
		f := root.FieldByIndex(key[:i+1])
		switch {
		case strings.IndexFunc(f.Name, unicode.IsLower) == -1: // All upper
			fallthrough
		case strings.LastIndexFunc(f.Name, unicode.IsUpper) == 0: // Initial cap
			b.WriteString(strings.ToLower(f.Name))
		default:
			for i, c := range f.Name {
				if unicode.IsUpper(c) {
					if i != 0 {
						b.WriteByte('-')
					}
					c = unicode.ToLower(c)
				}
				b.WriteRune(c)
			}
		}
		if i != len(sf.Index)-1 {
			b.WriteRune('.')
		}
	}
	return b.String()
}
