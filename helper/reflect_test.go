package helper

import (
	"encoding"
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnImplements(t *testing.T) {
	textUnmarshalerType := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

	for testName, itbl := range map[string]struct {
		seed          func() any
		postAssert    func(t testing.TB, v reflect.Value, err error)
		testTransform func(t testing.TB, input reflect.Value, v reflect.Value) (reflect.Value, error)
	}{
		"textUnmarshalerNonPointer": {
			seed: func() any { return net.ParseIP("10.1.1.1") },
			testTransform: func(t testing.TB, input reflect.Value, v reflect.Value) (reflect.Value, error) {
				inputIP, ok := input.Interface().(net.IP) // concrete net.IP
				assert.True(t, ok)
				assert.Equal(t, "10.1.1.1", inputIP.String())

				newType := v.Type()
				assert.Equal(t, reflect.PtrTo(reflect.TypeOf(net.IP{})), newType)

				outputIP, ok := v.Interface().(*net.IP)
				assert.True(t, ok)
				outputIP.UnmarshalText([]byte(`192.168.1.1`))

				return v, nil
			},
			postAssert: func(t testing.TB, v reflect.Value, err error) {
				assert.NoError(t, err)
				newIP, ok := v.Interface().(net.IP)
				assert.True(t, ok)
				assert.Equal(t, "192.168.1.1", newIP.String())
			},
		},
		"textUnmarshalerPointer": {
			seed: func() any {
				ip := net.ParseIP("10.1.1.1")
				return &ip
			},
			testTransform: func(t testing.TB, input reflect.Value, v reflect.Value) (reflect.Value, error) {
				inputIP, ok := input.Interface().(*net.IP) // note the pointer here
				assert.True(t, ok)
				assert.Equal(t, "10.1.1.1", inputIP.String())

				newType := v.Type()
				assert.Equal(t, reflect.PtrTo(reflect.TypeOf(net.IP{})), newType)

				outputIP, ok := v.Interface().(*net.IP)
				assert.True(t, ok)
				outputIP.UnmarshalText([]byte(`192.168.1.1`))

				return v, nil
			},
			postAssert: func(t testing.TB, v reflect.Value, err error) {
				assert.NoError(t, err)
				newIP, ok := v.Interface().(*net.IP)
				assert.True(t, ok)
				assert.Equal(t, "192.168.1.1", newIP.String())
			},
		},
		"nonTextUnmarshaler": {
			seed: func() any {
				// something explicitly NOT implementing TextUnmarshaler
				return net.UnixAddr{
					Name: "foo",
					Net:  "unix",
				}
			},
			testTransform: func(t testing.TB, _ reflect.Value, _ reflect.Value) (reflect.Value, error) {
				t.Fatalf("should not be invoked")
				return reflect.Value{}, nil
			},
			postAssert: func(t testing.TB, v reflect.Value, err error) {
				assert.NoError(t, err)
				vAddr, ok := v.Interface().(net.UnixAddr)
				assert.True(t, ok)
				assert.Equal(t, net.UnixAddr{Name: "foo", Net: "unix"}, vAddr)
			},
		},
	} {
		tbl := itbl
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			input := tbl.seed()
			newV, err := OnImplements(reflect.TypeOf(input), textUnmarshalerType, reflect.ValueOf(input), func(input reflect.Value, v reflect.Value) (reflect.Value, error) {
				return tbl.testTransform(t, input, v)
			})

			tbl.postAssert(t, newV, err)
		})
	}

}
