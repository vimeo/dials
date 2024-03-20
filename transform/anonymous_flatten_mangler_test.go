package transform

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vimeo/dials/common"
	"github.com/vimeo/dials/ptrify"
)

func TestTopLevelAnonymousFlatten(t *testing.T) {
	t.Parallel()
	type Config struct {
		unexposedHello string
		Hello          string
		Embed          `dials:"creative_name"`
		EmbedNoTag
	}

	c := &Config{
		unexposedHello: "hello world",
		Embed: Embed{
			Foo: "DoesThisWork",
		},
		EmbedNoTag: EmbedNoTag{
			World: "hello world",
		},
	}
	typeOfC := reflect.TypeOf(c)
	tVal := reflect.ValueOf(c)
	typeInstance := ptrify.Pointerify(typeOfC.Elem(), tVal.Elem())

	a := AnonymousFlattenMangler{}
	tfmr := NewTransformer(typeInstance, a)
	val, err := tfmr.Translate()
	require.NoError(t, err)

	expectedNames := []string{
		"Hello", "Foo", "Bar", "World",
	}

	expectedDialsTags := []string{
		"",
		"foofoo",
		"",
		"",
	}
	setFieldVals := []any{"fizzlebat", "foobar", true, "boop"}

	for i := 0; i < val.Type().NumField(); i++ {
		assert.Equal(t, expectedNames[i], val.Type().Field(i).Name)
		assert.EqualValues(t, expectedDialsTags[i], val.Type().Field(i).Tag.Get(common.DialsTagName))
		svIn := reflect.ValueOf(setFieldVals[i])
		sv := reflect.New(svIn.Type())
		sv.Elem().Set(svIn)
		val.Field(i).Set(sv)
	}
	revVal, revErr := tfmr.ReverseTranslate(val)
	if revErr != nil {
		t.Fatalf("failed to reverse translate value: %s", revErr)
	}
	expFieldVals := []any{"fizzlebat", []any{"foobar", true}, []any{"boop"}}
	for i, expVals := range expFieldVals {
		f := revVal.Field(i)
		if f.IsZero() {
			t.Errorf("field %d (%s) unexpectedly nil",
				i, revVal.Type().Field(i).Name)
			continue
		}
		if expInnerVals, ok := expVals.([]any); ok {
			// there's only one layer of fields, so don't try to be
			// fancy:
			f := f.Elem()
			for z, expVal := range expInnerVals {
				innerField := f.Field(z)
				if innerField.IsZero() {
					t.Errorf("inner field %d,%d (%s) unexpectedly nil",
						i, z, revVal.Type().Field(i).Type.Field(z).Name)
					continue
				}
				if innerField.Elem().Interface() != expVal {
					t.Errorf("unexpected value field %d.%d (%s.%s); got %v; want %v", i, z, revVal.Type().Field(i).Name, f.Type().Field(z).Name, innerField.Elem().Interface(), expVal)
				}
			}
		} else {
			if f.Elem().Interface() != expVals {
				t.Errorf("unexpected value field %d (%s); got %v; want %v", i, revVal.Type().Field(i).Name, f.Elem().Interface(), expVals)
			}
		}
	}
}

func TestTopLevelAnonymousFlattenWithNils(t *testing.T) {
	t.Parallel()
	type Config struct {
		unexposedHello string
		Hello          string
		// This field's tag has no place to go, so make sure it gets dropped
		Embed `dials:"creative_name"`
		EmbedNoTag
	}

	c := &Config{
		unexposedHello: "hello world",
		Embed: Embed{
			Foo: "DoesThisWork",
		},
		EmbedNoTag: EmbedNoTag{
			World: "hello world",
		},
	}
	typeOfC := reflect.TypeOf(c)
	tVal := reflect.ValueOf(c)
	typeInstance := ptrify.Pointerify(typeOfC.Elem(), tVal.Elem())

	a := AnonymousFlattenMangler{}
	tfmr := NewTransformer(typeInstance, a)
	val, err := tfmr.Translate()
	require.NoError(t, err)

	expectedNames := []string{
		"Hello", "Foo", "Bar", "World",
	}

	expectedDialsTags := []string{
		"",
		"foofoo",
		"",
		"",
	}
	setFieldVals := []any{"fizzlebat", "", true, ""}
	shouldNil := []bool{false, true, false, true}

	for i := 0; i < val.Type().NumField(); i++ {
		assert.Equal(t, expectedNames[i], val.Type().Field(i).Name)
		assert.EqualValues(t, expectedDialsTags[i], val.Type().Field(i).Tag.Get(common.DialsTagName))
		if shouldNil[i] {
			continue
		}
		svIn := reflect.ValueOf(setFieldVals[i])
		sv := reflect.New(svIn.Type())
		sv.Elem().Set(svIn)
		val.Field(i).Set(sv)
	}
	t.Logf("in: %+v", val.Interface())
	revVal, revErr := tfmr.ReverseTranslate(val)
	if revErr != nil {
		t.Fatalf("failed to reverse translate value: %s", revErr)
	}
	t.Logf("translated: %+v", revVal.Interface())
	expFieldVals := []any{"fizzlebat", []any{nil, true}, nil}
	expNil := []any{false, []bool{true, false}, true}
	for i, expVals := range expFieldVals {
		f := revVal.Field(i)
		if f.IsZero() {
			if expNil[i] == false {
				t.Errorf("field %d (%s) unexpectedly nil",
					i, revVal.Type().Field(i).Name)
			}
			continue
		}
		if expInnerVals, ok := expVals.([]any); ok {
			inExpNil := expNil[i].([]bool)
			// there's only one layer of fields, so don't try to be
			// fancy:
			f := f.Elem()
			for z, expVal := range expInnerVals {
				innerField := f.Field(z)
				if innerField.IsZero() {
					if !inExpNil[z] {
						t.Errorf("inner field %d,%d (%s.%s) unexpectedly nil",
							i, z, revVal.Type().Field(i).Name, revVal.Type().Field(i).Type.Elem().Field(z).Name)
					}
					continue
				}
				if innerField.Elem().Interface() != expVal {
					t.Errorf("unexpected value field %d.%d (%s.%s); got %v; want %v", i, z, revVal.Type().Field(i).Name, f.Type().Field(z).Name, innerField.Elem().Interface(), expVal)
				}
			}
		} else {
			if f.Elem().Interface() != expVals {
				t.Errorf("unexpected value field %d (%s); got %v; want %v", i, revVal.Type().Field(i).Name, f.Elem().Interface(), expVals)
			}
		}
	}
}
