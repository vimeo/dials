package dials

import (
	"errors"
	"fmt"
	"go/ast"
	"reflect"

	"github.com/vimeo/dials/ptrify"
)

var errCanSetField = errors.New("cannot set field")

func overlayField(base, overlay reflect.Value) error {
	switch overlay.Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Interface, reflect.Map:
		if overlay.IsNil() {
			return nil
		}
	default:
	}
	if !base.CanSet() {
		return errCanSetField
	}
	switch base.Kind() {
	case reflect.Ptr:
		// if we're dealing with a pointer in the original field, and
		// it's unset from lower layers, just set the pointer.
		if base.IsNil() {
			// make sure it's pointing to the same type
			if base.Type().Elem() == overlay.Type().Elem() {
				base.Set(overlay)
				//  we're done here
				return nil
			}
			// we've pointerified the field-type, so we need to
			// call overlayStruct.
			if base.Type().Elem().Kind() != reflect.Struct {
				return fmt.Errorf("unexpected kind for mangled pointer target: %s",
					base.Type().Elem().Kind())
			}
			if ptrify.IsTextUnmarshalerStruct(base.Type().Elem()) {
				return fmt.Errorf("unexpected shallow-copy-struct as pointer target types: base: %s; overlay %s",
					base.Type(), overlay.Type())
			}
			// We just need to allocate a new pointer to copy into,
			// then we'll be able to call overlayStruct.
			base.Set(reflect.New(base.Type().Elem()))
			return overlayStruct(base.Elem(), overlay.Elem())
		}
		if ptrify.IsTextUnmarshalerStruct(base.Type().Elem()) {
			// base is not nil and we're not deep-copying, so we can overwrite the pointer.
			if overlay.Type().AssignableTo(base.Type()) {
				base.Set(overlay)
			} else if overlay.Type().AssignableTo(base.Type().Elem()) {
				base.Elem().Set(overlay)
			}
			//  we're done here
			return nil
		}
		// both pointers are non-nil, and it's a pointerified struct.
		return overlayStruct(base.Elem(), overlay.Elem())
	case reflect.Interface:
		return overlayInterface(base, overlay)
	case reflect.Struct:
		if ptrify.IsTextUnmarshalerStruct(base.Type()) {
			// base is not nil and we're not deep-copying, so we can shallow-copy
			switch overlay.Kind() {
			case reflect.Ptr:
				base.Set(overlay.Elem())
			case reflect.Struct:
				if !overlay.Type().AssignableTo(base.Type()) {
					return fmt.Errorf("struct type %s is not assignable to %s",
						overlay.Type(), base.Type())
				}
				// it's assignable, just assign it.
				base.Set(overlay)
				return nil
			default:
				return fmt.Errorf("type %s is not assignable to %s",
					overlay.Type(), base.Type())
			}
			//  we're done here
			return nil
		}
		if overlay.Kind() == reflect.Ptr {
			// it's a pointerified struct.
			return overlayStruct(base, overlay.Elem())
		}
		return overlayStruct(base, overlay)
	default:
		// this probably will be a pointer to the value we want because
		// we explicitly pointerify fields, but there's a chance that
		// values coming through interface values might not be
		// pointer-ified (plus Sources can return whatever Value they
		// want)
		if overlay.Kind() == reflect.Ptr {
			base.Set(overlay.Elem())
		} else {
			base.Set(overlay)
		}
	}
	return nil
}

// overlayStruct assumes that overlay is a pointerified type of the type of
// base.
func overlayStruct(base, overlay reflect.Value) error {
	// panic since these violate the contract (in the function name).
	if base.Kind() != reflect.Struct {
		panic(fmt.Errorf("non-struct call: %s as base (%s as overlay)", base.Type(), overlay.Type()))
	}
	if overlay.Kind() != reflect.Struct {
		panic(fmt.Errorf("non-struct call: %s as overlay (%s as base)", overlay.Type(), base.Type()))
	}
	for i, j := 0, 0; i < base.NumField(); i++ {
		currentField := base.Field(i)
		if !ast.IsExported(base.Type().Field(i).Name) {
			continue
		}
		switch currentField.Kind() {
		// We skip channels and functions during
		// pointerification, so we need to adjust indices
		// appropriately and skip these fields.
		case reflect.Chan, reflect.Func:
			continue
		default:
		}
		if overlayErr := overlayField(
			currentField,
			overlay.Field(j)); overlayErr != nil {
			return fmt.Errorf("failed to set field %q (number %d): %s",
				base.Type().Field(i).Name, i, overlayErr)
		}
		// We only increment the offset into the
		// pointerfied/source-specific value if the field was present.
		j++
	}
	return nil
}

func overlayInterface(base, overlay reflect.Value) error {
	if base.Kind() != reflect.Interface {
		panic(fmt.Errorf("invalid base of kind %s as argument to overlayInterface; only Interface allowed",
			base.Kind()))
	}
	switch k := overlay.Kind(); k {
	case reflect.Interface:
		if overlay.IsNil() || (kindNilable(overlay.Elem().Kind()) && overlay.Elem().IsNil()) {
			// if overlay is nil then we're done here
			return nil
		}
		if base.IsNil() || (kindNilable(base.Elem().Kind()) && base.Elem().IsNil()) {
			// base is nil: just set it and declare victory
			//
			// Since pointerification doesn't convert to a concrete
			// type if the default value it gets is nil, this is
			// the common case for this case
			base.Set(overlay.Elem())
			return nil
		}
		if overlay.Elem().Type() == base.Elem().Type() {
			// they're the same underlying type, just
			// overlay-away (leave the base as an interface
			// so we fall into the next case)
			if err := overlayField(base, overlay.Elem()); err != nil {
				return fmt.Errorf("failed to overlay interface %s on interface %s (iface type %s): %s",
					overlay.Elem().Type(), base.Elem().Type(), base.Type(), err)
			}
			return nil
		}
		// interface values don't have the same type, and neither is
		// nil, treat it the same way as if base is nil, and just overlay.
		base.Set(overlay.Elem())
		return nil
	case reflect.Ptr:
		if overlay.IsNil() {
			// if overlay is nil then we're done here
			return nil
		}
		ptrImpl := overlay.Type().Implements(base.Type())
		baseImpl := overlay.Type().Elem().Implements(base.Type())
		if !ptrImpl && !baseImpl {
			return fmt.Errorf("overlay-type (%s) does not implement base type %s", overlay.Type(), base.Type())
		}
		// if we're here, we need to overlay within the same types
		// check whether the overlay pointer-type matches the base-value's contained type
		// or the pointee type matches the base-value's contained type
		if !base.IsNil() && (base.Elem().Type() == overlay.Type() || base.Elem().Type() == overlay.Type().Elem()) {
			if err := overlayField(base, overlay.Elem()); err != nil {
				return fmt.Errorf("failed to overlay ptr type %s onto %s: %s", overlay.Type(), base.Type(), err)
			}
			return nil
		}
		out := reflect.New(overlay.Type().Elem())
		if err := overlayField(out.Elem(), overlay.Elem()); err != nil {
			return fmt.Errorf("unable to overlay pointer %s onto interface %s: %s",
				overlay.Type(), base.Type(), err)
		}
		if ptrImpl {
			base.Set(out)
		} else if baseImpl {
			base.Set(out.Elem())
		} else {
			panic("unreachable code, neither ptr nor base-type implement interface")
		}
		return nil
	case reflect.Struct:
		if !base.IsNil() {
			if base.Elem().Type() == overlay.Type() || base.Elem().Type() == reflect.PtrTo(overlay.Type()) {
				out := reflect.New(base.Elem().Type())
				deepCopy(base.Elem(), out.Elem())
				if err := overlayField(out.Elem(), overlay); err != nil {
					return fmt.Errorf("failed to overlay interface onto struct field: %s", err)
				}
				base.Set(out.Elem())
				return nil
			}
		}
		if overlay.Type().Implements(base.Type()) {
			base.Set(overlay)
			return nil
		}
		if reflect.PtrTo(overlay.Type()).Implements(base.Type()) {
			out := reflect.New(overlay.Type())
			if err := overlayStruct(out.Elem(), overlay); err != nil {
				return fmt.Errorf("error overlaying struct(%s) onto interface(%s): %s",
					out.Type().Elem(), base.Type(), err)
			}
			base.Set(out)
			return nil
		}

	case reflect.Map, reflect.Slice:
		if overlay.IsNil() {
			return nil

		}
		// If it's a map or slice, then we can just overwrite
		// the overlay.
		if overlay.Type().Implements(base.Type()) {
			base.Set(overlay)
			return nil
		}
		return fmt.Errorf("error overlaying %s (%s) onto interface(%s); overlay doesn't implement interface-type",
			overlay.Elem().Kind(), overlay.Elem().Type(), base.Type())

	case reflect.Func, reflect.Chan:
		// this is an interface type, so we can be a little more liberal in how overlaying works.
		if overlay.IsNil() {
			return nil
		}
		if overlay.Type().Implements(base.Type()) {
			base.Set(overlay)
			return nil
		}
		return fmt.Errorf("error overlaying %s (%s) onto interface(%s); overlay doesn't implement interface-type",
			overlay.Elem().Kind(), overlay.Elem().Type(), base.Type())
	case reflect.Array:
		out := reflect.New(overlay.Type())
		if !base.IsNil() {
			if base.Elem().Type() == overlay.Type() {
				deepCopyArray(overlay, out.Elem())
				base.Set(out.Elem())
				return nil
			}
			if base.Elem().Type() == reflect.PtrTo(overlay.Type()) {
				deepCopyArray(overlay, out.Elem())
				base.Set(out)
				return nil
			}
		}
		if overlay.Type().Implements(base.Type()) {
			deepCopyArray(overlay, out.Elem())
			base.Set(out.Elem())
			return nil
		}
		if reflect.PtrTo(overlay.Type()).Implements(base.Type()) {
			deepCopyArray(overlay, out.Elem())
			base.Set(out)
			return nil
		}
		// fallthrough to the error below
	}
	// we somehow fell through here, which is weird
	return fmt.Errorf("fallthrough overlay interface type %s onto %s", overlay.Type(), base.Type())
}

func kindNilable(k reflect.Kind) bool {
	switch k {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return true
	default:
		return false
	}
}
