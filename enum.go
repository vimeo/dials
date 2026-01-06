//go:build go1.23

package dials

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ValueMap provides a mapping from a string to a type of your choice to be used
// like an enum.  Users should implement the `DialsValueMap()` method to provide
// all possible mappings values.  Any values not in the mapping will fail when
// unmarshaling.
type ValueMap[T any] interface {
	DialsValueMap() map[string]T
}

// Enum allows you to treat a certain type like an enumeration.  Enums are
// integrated into the flags packages to automatically amend the `Usage` text to
// include all allowed values.
type Enum[T ValueMap[T]] struct {
	Value T
}

// EnumValuable is an interface that is useful to get all the allowed enum
// values as strings.  It is used by the flag packages to adjust the usage text
// to include all available values.
type EnumValuable interface {
	AllowedEnumValues() []string
}

// AllowedEnumValues implements the [EnumValuable] interface.
func (e Enum[T]) AllowedEnumValues() []string {
	return slices.Sorted(maps.Keys(e.Value.DialsValueMap()))
}

// UnmarshalText implements [encoding.TextUnmarshaler] so that we can map from
// strings to our typed enum.  If there is no appropriate mapping,
// [ErrInvalidEnumValue] is returned.
func (e *Enum[T]) UnmarshalText(text []byte) error {
	allVals := e.Value.DialsValueMap()
	if mapping, ok := allVals[string(text)]; ok {
		e.Value = mapping
		return nil
	}
	return &ErrInvalidEnumValue{
		Input:   string(text),
		Allowed: e.AllowedEnumValues(),
	}
}

// FuzzyEnum is just like an [Enum], but does case-insensitive comparisons. Just
// like Enums, FuzzyEnums are integrated into the flags packages to
// automatically amend the `Usage` text to include all allowed values and will
// also indicate that the matching is case-insensitive.
type FuzzyEnum[T ValueMap[T]] struct {
	Value T
}

// FuzzyEnumComparer is an interface that allows us to detect that
// case-insensitive comparison has been used.
type FuzzyEnumComparer interface {
	isFuzzy()
}

// AllowedEnumValues implements the [EnumValuable] interface.
func (f FuzzyEnum[T]) AllowedEnumValues() []string {
	return slices.Sorted(maps.Keys(f.Value.DialsValueMap()))
}

// isFuzzy implements the FuzzyEnumComparer interface.
func (f FuzzyEnum[T]) isFuzzy() {}

// UnmarshalText implements [encoding.TextUnmarshaler] and does a case-insensitive
// comparison of the string to map back to the appropriate value from the
// enumeration.  If there is no appropriate mapping, [ErrInvalidEnumValue] is
// returned.
func (f *FuzzyEnum[T]) UnmarshalText(text []byte) error {
	allVals := f.Value.DialsValueMap()
	noCase := make(map[string]T, len(allVals))
	for k, v := range allVals {
		noCase[strings.ToLower(k)] = v
	}

	if mapping, ok := noCase[strings.ToLower(string(text))]; ok {
		f.Value = mapping
		return nil
	}

	return &ErrInvalidEnumValue{
		Input:   string(text),
		Allowed: f.AllowedEnumValues(),
		Fuzzy:   true,
	}
}

// ErrInvalidEnumValue is an error that is returned when there is no mapping for
// a particular value in the enumeration.
type ErrInvalidEnumValue struct {
	// All the allowed inputs.
	Allowed []string
	// The input that was provided (not in the Allowed list).
	Input string
	// true if the comparison is case-insensitive.
	Fuzzy bool
}

// Error implements the error interface.
func (e *ErrInvalidEnumValue) Error() string {
	return fmt.Sprintf("value %q is not part of the allowed enumeration: %+v, fuzzy: %t", e.Input, e.Allowed, e.Fuzzy)
}

// StringValueMap is a helper that converts string-ish typed-enums to the
// mapping table expected by the [ValueMap] interface.
func StringValueMap[T ~string](in ...T) map[string]T {
	mapping := make(map[string]T, len(in))
	for _, element := range in {
		mapping[string(element)] = element
	}
	return mapping
}

// StringerValueMap is a helper that converts types that implement the
// [fmt.Stringer] interface to the mapping table expected by the [ValueMap]
// interface.
func StringerValueMap[T fmt.Stringer](in ...T) map[string]T {
	mapping := make(map[string]T, len(in))
	for _, element := range in {
		mapping[element.String()] = element
	}
	return mapping
}

// EnumValue is a helper that's particularly useful when setting enum defaults in configuration.
func EnumValue[T ValueMap[T]](in T) Enum[T] {
	return Enum[T]{
		Value: in,
	}
}
