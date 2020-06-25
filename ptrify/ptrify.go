package ptrify

import (
	"encoding"
	"go/ast"
	"reflect"

	"github.com/vimeo/dials/common"
)

// Note: this looks weird because it is, you need to call TypeOf on a nil
// pointer here then take the element type, otherwise you get a nil type and
// that's not useful (it actually generates a panic when it's used further down).
var textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// Pointerify takes a type and returns another type with all its members
// set to pointers of their respective types
func Pointerify(original reflect.Type, tmpl reflect.Value) reflect.Type {
	newFields := make([]reflect.StructField, 0, original.NumField())

	for i := 0; i < original.NumField(); i++ {
		originalField := original.Field(i)
		var tmplFieldVal reflect.Value
		if tmpl.IsValid() {
			tmplFieldVal = tmpl.Field(i)
		}

		if OmitField(originalField) {
			continue
		}

		sf := pointerifyField(originalField, tmplFieldVal)
		if sf != nil {
			newFields = append(newFields, *sf)
		}
	}

	return reflect.StructOf(newFields)
}

// OmitField returns a boolean indicating whether the field should be skipped
// because the dials tag value is "-" (`dials:"-"`) or because the field is
// unexported
func OmitField(sf reflect.StructField) bool {

	if !ast.IsExported(sf.Name) {
		// reflect.StructOf panics on unexported fields, skip
		// them.
		return true
	}

	if dtv, ok := sf.Tag.Lookup(common.DialsTagName); ok && dtv == "-" {
		// ignore the fields with "-" tags (ex: `dials:"-"`)
		return true
	}

	return false

}

func pointerifyField(originalField reflect.StructField, tmplFieldVal reflect.Value) *reflect.StructField {
	ft := originalField.Type
	sf := reflect.StructField{
		Name:      originalField.Name,
		Type:      reflect.PtrTo(ft),
		Tag:       originalField.Tag,
		PkgPath:   originalField.PkgPath,
		Anonymous: originalField.Anonymous,
	}
	switch ft.Kind() {
	case reflect.Map, reflect.Slice:
		// These are already nil-able, so use the field rather
		// than the pointer-ized field.
		return &originalField
	case reflect.Interface:
		// check whether the field is nil in the
		// template/default struct. (we'll devirtualize down to
		// the concrete type if it is there).
		if !tmplFieldVal.IsValid() || tmplFieldVal.IsNil() {
			// If it's nil, we have to preserve the original field
			// as-is.
			return &originalField
		}
		impl := tmplFieldVal.Elem()
		switch impl.Kind() {
		// nil-able types, which we'd take as-is anyway.
		case reflect.Map, reflect.Slice:
			// override the type to the concrete type supplied.
			sf.Type = impl.Type()
			return &sf
		case reflect.Array:
			// override the type to a pointer to the concrete type supplied.
			sf.Type = reflect.PtrTo(impl.Type())
			return &sf
		case reflect.Chan, reflect.Func:
			// these are not useful types for config, but maybe
			// there are some other implementations of the
			// interface that Sources know about.
			return &originalField
		case reflect.Ptr, reflect.Struct:
			newSF := originalField
			newSF.Type = impl.Type()
			return pointerifyField(newSF, impl)
		}
		return &originalField
	case reflect.Ptr:
		// These are already pointers or nil-able, so use the
		// field rather than the pointer-ized field.
		if ft.Elem().Kind() != reflect.Struct {
			return &originalField
		}
		// it's a pointer to a struct, so we need to
		// pointerize the pointee, just update the type and
		// fallthrough
		sf.Type = ft
		ft = ft.Elem()
		// if it's a valid pointer that's non-nil, dereference it so it
		// can be used in the fallthrough
		if tmplFieldVal.Kind() == reflect.Ptr && !tmplFieldVal.IsNil() {
			tmplFieldVal = tmplFieldVal.Elem()
		} else {
			tmplFieldVal = reflect.Value{}
		}
		fallthrough
	case reflect.Struct:
		// first check whether this implements
		// `TextUnmarshaler`, in which case we'll pointerify
		// this field and leave its type alone.
		if IsTextUnmarshalerStruct(ft) {
			return &sf
		}
		// It's a struct without an UnmarshalText method, we
		// need to recursively pointerify the component fields.
		pointeredStruct := Pointerify(ft, tmplFieldVal)
		return &reflect.StructField{
			Name:      originalField.Name,
			Type:      reflect.PtrTo(pointeredStruct),
			Tag:       originalField.Tag,
			Anonymous: originalField.Anonymous,
		}
	case reflect.Chan, reflect.Func:
		// channels are not configuration, and defaulting to
		// setting the capacity based on the config value would
		// be a bit too much magic.
		// functions are not configuration at all.
		// Skip this field
		return nil
	default:
		return &sf
	}
}

// IsTextUnmarshalerStruct indicates whether a struct-type implements
// encoding.TextUnmarshaler either directly or via its pointer-type
func IsTextUnmarshalerStruct(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	return (t.Implements(textUnmarshaler) ||
		reflect.PtrTo(t).Implements(textUnmarshaler))
}
