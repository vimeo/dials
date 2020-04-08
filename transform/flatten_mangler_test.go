package transform

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vimeo/dials/ptrify"
)

func TestMangler(t *testing.T) {
	type foo struct {
		Location string
	}

	type bar struct {
		Name   string
		Foobar *foo
	}

	b := bar{
		Name: "test",
		Foobar: &foo{
			Location: "here",
		},
	}

	btype := reflect.TypeOf(b)

	sf := reflect.StructField{Name: "ConfigField", Type: btype}
	configStructType := reflect.StructOf([]reflect.StructField{sf})

	ptrifiedConfigType := ptrify.Pointerify(configStructType, reflect.New(configStructType).Elem())

	f := &FlattenMangler{}

	tfmr := NewTransformer(ptrifiedConfigType, f)
	val, err := tfmr.Translate()
	require.NoError(t, err)
	fmt.Println("val", val)
	// f.walk("", []int{}, bv, bt)
	// fmt.Println("fieldPaths", f.fieldPaths)
	// fmt.Println(f.fieldPaths)
}

func TestRandom(t *testing.T) {

}
