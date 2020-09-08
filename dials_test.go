package dials

import (
	"context"
	"errors"
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

func TestConfigWithoutVerifier(t *testing.T) {
	t.Parallel()
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

// failVerifier is a struct with a Verify() method that always fails with an error
type failVerifier struct{}

var errFailVerifier = errors.New("fail")

func (failVerifier) Verify() error {
	return errFailVerifier
}

var _ VerifiedConfig = (*failVerifier)(nil)

func TestConfigWithFailVerifier(t *testing.T) {
	t.Parallel()
	type testConfig struct {
		failVerifier
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
	_, err := Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
	if !errors.Is(err, errFailVerifier) {
		t.Fatalf("unexpected error: got %s; expected %s", err, errFailVerifier)
	}
}

// successVerifier is a struct with a Verify() method that always fails with an error
type successVerifier struct{}

func (successVerifier) Verify() error {
	return nil
}

var _ VerifiedConfig = (*successVerifier)(nil)

func TestConfigWithSuccessVerifier(t *testing.T) {
	t.Parallel()
	type testConfig struct {
		successVerifier
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

// configurableVerifier is a struct with a Verify() method that fails depending
// on the value of valid.
type configurableVerifier struct {
	Valid bool
	Foo   string
}

func (c configurableVerifier) Verify() error {
	if c.Valid {
		return nil
	}
	return errFailVerifier
}

var _ VerifiedConfig = (*configurableVerifier)(nil)

func TestConfigWithConfigureVerifier(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false

	type ptrifiedConfig struct {
		Valid *bool
		Foo   *string
	}

	base := configurableVerifier{
		Valid: true,
		Foo:   "foo",
	}
	emptyConf := ptrifiedConfig{
		Valid: nil,
		Foo:   nil,
	}
	// Push a new value, that should overlay on top of the base
	foozleStr := "foozle"
	foozleConfig := ptrifiedConfig{
		Valid: &trueVal,
		Foo:   &foozleStr,
	}

	// setup a cancelable context so the monitor goroutine gets shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	params := Params{
		OnWatchedError: func(ctx context.Context, err error, _, _ interface{}) { errCh <- err },
	}

	w := fakeWatchingSource{fakeSource: fakeSource{outVal: foozleConfig}}
	d, err := params.Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
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
	select {
	case c := <-d.Events():
		assert.Equal(t, "fim", c.(*configurableVerifier).Foo)
		assert.Equal(t, "fim", d.View().(*configurableVerifier).Foo)
	case err := <-errCh:
		t.Errorf("unexpected error from monitor: %s", err)
	}

	// send a config with Valid set to false
	invalidStr := "invalid"
	invalidConfig := ptrifiedConfig{Valid: &falseVal, Foo: &invalidStr}
	w.send(ctx, reflect.ValueOf(invalidConfig))
	select {
	case unexpectedConf := <-d.Events():
		assert.Equal(t, "foo", unexpectedConf.(*configurableVerifier).Foo)
		assert.Equal(t, "foo", d.View().(*configurableVerifier).Foo)
		assert.False(t, unexpectedConf.(*configurableVerifier).Valid)
		assert.False(t, d.View().(*configurableVerifier).Valid)
	case err := <-errCh:
		if !errors.Is(err, errFailVerifier) {
			t.Errorf("unexpected error from verification failure: %s", err)
		}
	}

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	select {
	case finalConf := <-d.Events():
		assert.Equal(t, "foo", finalConf.(*configurableVerifier).Foo)
		assert.Equal(t, "foo", d.View().(*configurableVerifier).Foo)
	case err := <-errCh:
		t.Errorf("unexpected error from monitor: %s", err)
	}
}
