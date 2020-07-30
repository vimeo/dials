package dials

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	type testConfig struct {
		Foo string
	}

	tc := testConfig{
		Foo: "foo",
	}

	// no sources, just passing the defaults
	d, err := Config(context.Background(), &tc)
	require.NoError(t, err)

	filledConfig := &testConfig{}
	d.Fill(filledConfig)
	assert.Equal(t, "foo", filledConfig.Foo)
}

type fakeSource struct {
	outVal interface{}
}

func (f *fakeSource) Value(t *Type) (reflect.Value, error) {
	return reflect.ValueOf(f.outVal).Convert(t.t), nil
}

type fakeWatchingSource struct {
	fakeSource
	t  *Type
	cb func(context.Context, reflect.Value)
}

func (f *fakeWatchingSource) Watch(_ context.Context, t *Type, cb func(context.Context, reflect.Value)) error {
	f.cb = cb
	f.t = t
	return nil
}

func (f *fakeWatchingSource) send(ctx context.Context, val reflect.Value) {
	f.cb(ctx, val.Convert(f.t.t))
}

func TestConfig(t *testing.T) {
	type testConfig struct {
		Foo string
	}

	type ptrifiedConfig struct {
		Foo *string
	}

	base := testConfig{
		Foo: "foo",
	}
	emptyConf := ptrifiedConfig{
		Foo: nil,
	}
	// Push a new value, that should overlay on top of the base
	foozleStr := "foozle"
	foozleConfig := ptrifiedConfig{
		Foo: &foozleStr,
	}

	// setup a cancelable context so the monitor goroutine gets shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := fakeWatchingSource{fakeSource: fakeSource{outVal: foozleConfig}}
	d, err := Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
	require.NoError(t, err)

	// pull in the existing value and verify that it works as intended.
	// overwrite our original "base" to verify deep-copying is doing its job.
	d.Fill(&base)
	assert.Equal(t, "foozle", base.Foo)

	// Push a new value, that should overlay on top of the base
	fimStr := "fim"
	fimConfig := ptrifiedConfig{
		Foo: &fimStr,
	}
	w.send(ctx, reflect.ValueOf(fimConfig))
	c := <-d.Events()
	assert.Equal(t, "fim", c.(*testConfig).Foo)
	assert.Equal(t, "fim", d.View().(*testConfig).Foo)

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-d.Events()
	assert.Equal(t, "foo", finalConf.(*testConfig).Foo)
	assert.Equal(t, "foo", d.View().(*testConfig).Foo)
}
