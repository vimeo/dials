package transform

import (
	"fmt"
	"reflect"
)

// SingleTypeSubstitutionMangler implements Mangler, and converts between two types that are convertible to one another
// Must be constructed with NewSingleTypeSubstitutionMangler to fill in the unexported reflect.Type fields.
// The two types must be convertible (by Go's definition), so we can directly convert one value to the other.
// The F type-argument is the "from" type, which is being being replaced.
// The T type-argument is the "to" type, which is taking its place during processing (and swapped back to F during Unmangle).
type SingleTypeSubstitutionMangler[F, T any] struct {
	from, to reflect.Type
}

// NewSingleTypeSubstitutionMangler constructs a new SingleTypeSubstitutionMangler, filling in the unexported fields.
// The F type-argument is the "from" type, which is being being replaced.
// The T type-argument is the "to" type, which is taking its place during processing (and swapped back to F during Unmangle).
// If T is not convertible to F, an error is returned. (only Unmangle converts
// values back, so convertibility in the other direction is irrelevant)
func NewSingleTypeSubstitutionMangler[F, T any]() (*SingleTypeSubstitutionMangler[F, T], error) {
	from := reflect.TypeFor[F]()
	to := reflect.TypeFor[T]()
	if !to.ConvertibleTo(from) {
		return nil, fmt.Errorf("type %s is not convertible to %s", to, from)
	}
	return &SingleTypeSubstitutionMangler[F, T]{
		from: from,
		to:   to,
	}, nil
}

// Mangle is called for every field in a struct, and maps that to one or more output fields.
// Implementations that desire to leave fields unchanged should return
// the argument unchanged. (particularly useful if taking advantage of
// recursive evaluation)
func (s *SingleTypeSubstitutionMangler[F, T]) Mangle(sf reflect.StructField) ([]reflect.StructField, error) {
	if newType, sub := s.subType(sf.Type); sub {
		newField := sf
		newField.Type = newType

		return []reflect.StructField{newField}, nil
	}
	// nothing to do
	return []reflect.StructField{sf}, nil
}

func (s *SingleTypeSubstitutionMangler[F, T]) subType(t reflect.Type) (reflect.Type, bool) {
	if t == s.from {
		return s.to, true
	}
	if t == reflect.PointerTo(s.from) {
		return reflect.PointerTo(s.to), true
	}
	// With the easy cases out of the way, handle maps, arrays and slices
	switch t.Kind() {
	case reflect.Pointer:
		nElem, subPtr := s.subType(t.Elem())
		if !subPtr {
			return t, false
		}
		return reflect.PointerTo(nElem), true
	case reflect.Map:
		nkType, subKey := s.subType(t.Key())
		nvType, subVal := s.subType(t.Elem())
		if !subKey && !subVal {
			return t, false
		}
		return reflect.MapOf(nkType, nvType), true
	case reflect.Array:
		nElem, subArray := s.subType(t.Elem())
		if !subArray {
			return t, false
		}
		return reflect.ArrayOf(t.Len(), nElem), true
	case reflect.Slice:
		nElem, subSlice := s.subType(t.Elem())
		if !subSlice {
			return t, false
		}
		return reflect.SliceOf(nElem), true
	case reflect.Chan:
		nElem, subChan := s.subType(t.Elem())
		if !subChan {
			return t, false
		}
		return reflect.ChanOf(t.ChanDir(), nElem), true
	default:
		// All other types are handled by the Transformer recursing
		return t, false
	}
}

// Unmangle is called for every source-field->mangled-field
// mapping-set, with the mangled-field and its populated value set. The
// implementation of Unmangle should return a reflect.Value that will
// be used for the next mangler or final struct value)
// Returned reflect.Value should be convertible to the field's type.
func (s *SingleTypeSubstitutionMangler[F, T]) Unmangle(sf reflect.StructField, fval []FieldValueTuple) (reflect.Value, error) {
	// fortunately, we never drop fields, or split fields out, so this is easy :)
	out, _ := s.subVal(sf.Type, fval[0].Value)
	return out, nil
}

func (s *SingleTypeSubstitutionMangler[F, T]) subVal(t reflect.Type, mVal reflect.Value) (reflect.Value, bool) {
	if t == s.from {
		return mVal.Convert(s.from), true
	}
	if t == reflect.PointerTo(s.from) {
		// it'll be a pointer
		if mVal.IsNil() {
			return reflect.Zero(t), true
		}
		outVal := mVal.Elem().Convert(s.from)
		outPtr := reflect.New(s.from)
		outPtr.Elem().Set(outVal)
		return outPtr, true
	}
	// With the easy cases out of the way, handle maps, arrays and slices
	// first check for nil pointers/maps/slices/channels, though
	if isNil(mVal) {
		// Return a nil of the right type.
		// There's nothing to do.
		return reflect.Zero(t), true
	}
	// skip the rest if subType wouldn't substitute anything
	_, subArray := s.subType(t)
	if !subArray {
		return mVal, false
	}
	switch t.Kind() {
	case reflect.Pointer:
		nElem, subPtr := s.subVal(t.Elem(), mVal.Elem())
		if !subPtr {
			return mVal, false
		}
		return nElem.Addr(), true
	case reflect.Map:
		// we mangled the map type, and the map value we're converting back is non-nil
		out := reflect.MakeMapWithSize(t, mVal.Len())
		mIter := mVal.MapRange()
		for mIter.Next() {
			k, _ := s.subVal(t.Key(), mIter.Key())
			v, _ := s.subVal(t.Elem(), mIter.Value())
			out.SetMapIndex(k, v)
		}
		return out, true
	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := 0; i < mVal.Len(); i++ {
			slot := mVal.Index(i)
			nVal, _ := s.subVal(t.Elem(), slot)
			out.Index(i).Set(nVal)
		}
		return out, true
	case reflect.Slice:
		out := reflect.MakeSlice(t, mVal.Len(), mVal.Cap())
		for i := 0; i < mVal.Len(); i++ {
			slot := mVal.Index(i)
			nVal, _ := s.subVal(t.Elem(), slot)
			out.Index(i).Set(nVal)
		}
		return out, true
	case reflect.Chan:
		directionlessT := reflect.ChanOf(reflect.BothDir, t.Elem())
		out := reflect.MakeChan(directionlessT, mVal.Cap())
		if mVal.Cap() == 0 {
			return out, true
		}
		// Since we constructed these two channels to have the same
		// capacity, and there shouldn't be a way for anything else to
		// be sending on this channel, we can copy any values into the final one.
		for {
			_, recvVal, _ := reflect.Select([]reflect.SelectCase{{Dir: reflect.SelectRecv, Chan: mVal}, {Dir: reflect.SelectDefault}})
			if !recvVal.IsValid() {
				// break out of the loop when we empty the channel
				break
			}
			nVal, _ := s.subVal(out.Type().Elem(), recvVal)
			out.Send(nVal) // this shouldn't block unless some blockhead is still sending on the channel
		}
		return out, true
	default:
		// All other types are handled by the Transformer recursing
		return mVal, false
	}
}

// ShouldRecurse is called after Mangle for each field so nested struct
// fields get iterated over after any transformation done by Mangle().
func (s *SingleTypeSubstitutionMangler[F, T]) ShouldRecurse(reflect.StructField) bool {
	return true
}
