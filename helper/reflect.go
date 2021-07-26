package helper

import (
	"fmt"
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

	wasPointer := false

	if t.Kind() == reflect.Ptr {
		wasPointer = true
		t = t.Elem()
	}

	fmt.Printf("type %v implements iface %v %t\n", t, iface, t.Implements(iface))
	fmt.Printf("type %v implements iface %v %t\n", reflect.PtrTo(t), iface, reflect.PtrTo(t).Implements(iface))

	switch {
	case t.Implements(iface):
		implemented = implementsAsConcrete
		newVal = reflect.New(t).Elem()
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

	if v.IsNil() {
		if wasPointer {
			fmt.Printf("returning zero: %v\n", reflect.Zero(reflect.PtrTo(t)).Type())
			return reflect.Zero(reflect.PtrTo(t)), nil
		}
		fmt.Printf("it wasn't a pointer\n")
		return reflect.Zero(t), nil
	}

	if implemented == implementsAsPointer && !wasPointer {
		return v.Elem(), nil
	}

	return v, nil
}
