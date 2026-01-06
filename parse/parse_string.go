package parse

import (
	"fmt"
	"reflect"
	"strconv"
)

// String casts the provided string into the provided type, returning the
// result in a reflect.Value.
func String(str string, t reflect.Type) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(&str), nil
	case reflect.Bool:
		converted, err := strconv.ParseBool(str)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(&converted), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return parseNumber(str, t)
	case reflect.Slice:
		converted, err := StringSlice(str)
		if err != nil {
			return reflect.Value{}, err
		}
		convertedVal := reflect.ValueOf(converted)
		if convertedVal.Type() == t {
			return convertedVal, nil
		}
		castSlice := reflect.MakeSlice(t, 0, len(converted))
		for idx, strVal := range converted {
			castVal, parseErr := String(strVal, t.Elem())
			if parseErr != nil {
				return reflect.Value{}, fmt.Errorf("parse error of item %d %q: %s", idx, strVal, parseErr)
			}
			castSlice = reflect.Append(castSlice, castVal.Elem())
		}
		return castSlice, nil

	case reflect.Map:
		switch t {
		case reflect.TypeFor[map[string][]string]():
			converted, err := StringStringSliceMap(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(converted), nil
		case reflect.TypeOf(map[string]struct{}{}):
			converted, err := StringSet(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(converted), nil
		default:
			keyKind := t.Key().Kind()
			valKind := t.Elem().Kind()
			err := checkKindsSupported(keyKind, valKind)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("unsupported map type: %v", t)
			}
			converted, err := Map(str, t)
			if err != nil {
				return reflect.Value{}, err
			}
			return converted, nil
		}
	default:
		// If the type of the original StructField is unsupported, return an error.
		return reflect.Value{}, fmt.Errorf("value %q cannot be translated to kind %q", str, t.Kind())
	}
}

func checkKindsSupported(kinds ...reflect.Kind) error {
	for _, k := range kinds {
		switch k {
		case reflect.String, reflect.Bool, reflect.Int, reflect.Float64,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Float32, reflect.Complex64,
			reflect.Complex128:
			// no-op
		default:
			return fmt.Errorf("kind %v not supported", k)
		}
	}
	return nil
}
