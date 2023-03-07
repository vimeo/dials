package parse

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

var durationType = reflect.TypeOf(time.Duration(0))

func parseNumber(strVal string, numberType reflect.Type) (reflect.Value, error) {
	var castVal reflect.Value

	switch numberType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if numberType == durationType {
			convertedDuration, err := time.ParseDuration(strVal)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(&convertedDuration), nil
		}

		converted, err := strconv.ParseInt(strVal, 0, 64)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowInt(converted) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		switch numberType.Kind() {
		case reflect.Int:
			convertedInt := int(converted)
			castVal = reflect.ValueOf(&convertedInt)
		case reflect.Int8:
			converted8 := int8(converted)
			castVal = reflect.ValueOf(&converted8)
		case reflect.Int16:
			converted16 := int16(converted)
			castVal = reflect.ValueOf(&converted16)
		case reflect.Int32:
			converted32 := int32(converted)
			castVal = reflect.ValueOf(&converted32)
		case reflect.Int64:
			converted64 := int64(converted)
			castVal = reflect.ValueOf(&converted64)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		converted, err := strconv.ParseUint(strVal, 0, 64)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowUint(converted) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		switch numberType.Kind() {
		case reflect.Uint:
			uintConverted := uint(converted)
			castVal = reflect.ValueOf(&uintConverted)
		case reflect.Uint8:
			uintConverted := uint8(converted)
			castVal = reflect.ValueOf(&uintConverted)
		case reflect.Uint16:
			uintConverted := uint16(converted)
			castVal = reflect.ValueOf(&uintConverted)
		case reflect.Uint32:
			uintConverted := uint32(converted)
			castVal = reflect.ValueOf(&uintConverted)
		case reflect.Uint64:
			uintConverted := uint64(converted)
			castVal = reflect.ValueOf(&uintConverted)
		}
	case reflect.Float32:
		converted, err := strconv.ParseFloat(strVal, 32)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowFloat(converted) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		fl32 := float32(converted)
		castVal = reflect.ValueOf(&fl32)
	case reflect.Float64:
		converted, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowFloat(converted) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		castVal = reflect.ValueOf(&converted)
	case reflect.Complex64:
		converted, err := Complex64(strVal)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowComplex(complex128(converted)) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		castVal = reflect.ValueOf(&converted)
	case reflect.Complex128:
		converted, err := Complex128(strVal)
		if err != nil {
			return reflect.Value{}, &NumberError{err: err}
		}

		// Check for overflow
		convertTo := reflect.Zero(numberType)
		if convertTo.OverflowComplex(converted) {
			return reflect.Value{}, &OverflowError{err: fmt.Errorf("overflow of %v type: %v", numberType, converted)}
		}

		castVal = reflect.ValueOf(&converted)
	}
	return castVal, nil
}

// OverflowError represents an overflow when casting to a numeric type.
type OverflowError struct {
	err error
}

func (e *OverflowError) Unwrap() error {
	return e.err
}

func (e *OverflowError) Error() string {
	return e.err.Error()
}

// NumberError represents an error when parsing a string to generate a numeric type.
type NumberError struct {
	err error
}

func (e *NumberError) Error() string {
	return e.err.Error()
}

func (e *NumberError) Unwrap() error {
	return e.err
}
