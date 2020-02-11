package parsestring

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	stringSliceType     = reflect.TypeOf([]string{})
	boolSliceType       = reflect.TypeOf([]bool{})
	intSliceType        = reflect.TypeOf([]int{})
	int8SliceType       = reflect.TypeOf([]int8{})
	int16SliceType      = reflect.TypeOf([]int16{})
	int32SliceType      = reflect.TypeOf([]int32{})
	int64SliceType      = reflect.TypeOf([]int64{})
	uintSliceType       = reflect.TypeOf([]uint{})
	uint8SliceType      = reflect.TypeOf([]uint8{})
	uint16SliceType     = reflect.TypeOf([]uint16{})
	uint32SliceType     = reflect.TypeOf([]uint32{})
	float32SliceType    = reflect.TypeOf([]float32{})
	float64SliceType    = reflect.TypeOf([]float64{})
	complex64SliceType  = reflect.TypeOf([]complex64{})
	complex128SliceType = reflect.TypeOf([]complex128{})
)

// ParseString casts the provided string into the provided type, returning the
// result in a reflect.Value.
func ParseString(str string, t reflect.Type) (reflect.Value, error) {
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
		switch t {
		case stringSliceType:
			converted, err := ParseStringSlice(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(converted), nil
		case boolSliceType, intSliceType, int8SliceType, int16SliceType,
			int32SliceType, int64SliceType, uintSliceType, uint8SliceType,
			uint16SliceType, uint32SliceType, float32SliceType, float64SliceType,
			complex64SliceType, complex128SliceType:
			strVals := strings.Split(str, ",")
			castSlice := reflect.MakeSlice(t, 0, len(strVals))

			for _, strVal := range strVals {
				castVal, err := ParseString(strings.TrimSpace(strVal), t)
				if err != nil {
					return reflect.Value{}, err
				}
				castSlice = reflect.Append(castSlice, castVal.Elem())
			}

			return castSlice, nil
		default:
			return reflect.Value{}, fmt.Errorf("Unsupported slice type: %+v", t)
		}
	case reflect.Map:
		switch t {
		case reflect.TypeOf(map[string][]string{}):
			converted, err := ParseStringStringSliceMap(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(converted), nil
		case reflect.TypeOf(map[string]struct{}{}):
			converted, err := ParseStringSet(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(converted), nil
		default:
			keyKind := t.Key().Kind()
			valKind := t.Elem().Kind()
			err := checkKindsSupported(keyKind, valKind)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("Unsupported map type: %v", t)
			}
			converted, err := ParseMap(str, t)
			if err != nil {
				return reflect.Value{}, err
			}
			return converted, nil
		}
	default:
		// If the type of the original StructField is unsupported, return an error.
		return reflect.Value{}, fmt.Errorf("Value %q cannot be translated to kind %q", str, t.Kind())
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
			return fmt.Errorf("Kind %v not supported", k)
		}
	}
	return nil
}
