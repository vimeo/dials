package helper

import (
	"reflect"
)

const (
	doesNotImplement = iota
	implementsAsConcrete
	implementsAsPointer
)

// TransformFunc is a function that runs when a particular reflect.Type
// implements an interface.
type TransformFunc func(input reflect.Value, v reflect.Value) (reflect.Value, error)

// OnImplements automatically negotiates between pointered and concrete types.
// `t` is the source type. `iface` is the interface type we're checking. `input`
// is the seed value that will be operated on if t implements iface in one form
// or another and returned unmodified if it doesn't implement the interface.
// `op` is the operation that will be run if `t` implements `iface`.  It is
// passed the input, and a new value in the proper form of `t` and should return
// a modified value and an error.  This function returns the proper form of the
// modified value or the original unmodified input.
func OnImplements(t reflect.Type, iface reflect.Type, input reflect.Value, op TransformFunc) (reflect.Value, error) {
	implemented := doesNotImplement
	var newVal reflect.Value

	switch {
	case t.Implements(iface):
		implemented = implementsAsConcrete
		newVal = reflect.New(t.Elem())
	case reflect.PtrTo(t).Implements(iface):
		implemented = implementsAsPointer
		newVal = reflect.New(t)
	default:
		return input, nil
	}

	v, err := op(input, newVal)
	if err != nil {
		return input, err
	}

	if implemented == implementsAsPointer {
		return v.Elem(), nil
	}

	return v, nil
}
