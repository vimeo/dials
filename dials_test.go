package dials

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"testing"
	"time"

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

func (f *fakeSource) Value(_ context.Context, t *Type) (reflect.Value, error) {
	return reflect.ValueOf(f.outVal).Convert(t.t), nil
}

type fakeWatchingSource struct {
	fakeSource
	t    *Type
	args WatchArgs
}

func (f *fakeWatchingSource) Watch(_ context.Context, t *Type, args WatchArgs) error {
	f.args = args
	f.t = t
	return nil
}

func (f *fakeWatchingSource) send(ctx context.Context, val reflect.Value) {
	f.args.ReportNewValue(ctx, val.Convert(f.t.t))
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
	assert.Equal(t, "fim", c.Foo)
	assert.Equal(t, "fim", d.View().Foo)

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-d.Events()
	assert.Equal(t, "foo", finalConf.Foo)
	assert.Equal(t, "foo", d.View().Foo)
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

func TestConfigWithSkippedInitialVerify(t *testing.T) {
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
	_, err := Params[testConfig]{
		SkipInitialVerification: true,
	}.Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
	assert.NoError(t, err)
}

// successVerifier is a struct with a Verify() method that never fails with an error
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
	assert.Equal(t, "fim", c.Foo)
	assert.Equal(t, "fim", d.View().Foo)

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-d.Events()
	assert.Equal(t, "foo", finalConf.Foo)
	assert.Equal(t, "foo", d.View().Foo)
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
	params := Params[configurableVerifier]{
		OnWatchedError: func(ctx context.Context, err error, _, _ *configurableVerifier) { errCh <- err },
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
		assert.Equal(t, "fim", c.Foo)
		assert.Equal(t, "fim", d.View().Foo)
	case err := <-errCh:
		t.Errorf("unexpected error from monitor: %s", err)
	}

	// send a config with Valid set to false
	invalidStr := "invalid"
	invalidConfig := ptrifiedConfig{Valid: &falseVal, Foo: &invalidStr}
	w.send(ctx, reflect.ValueOf(invalidConfig))
	select {
	case unexpectedConf := <-d.Events():
		assert.Equal(t, "foo", unexpectedConf.Foo)
		assert.Equal(t, "foo", d.View().Foo)
		assert.False(t, unexpectedConf.Valid)
		assert.False(t, d.View().Valid)
	case err := <-errCh:
		if !errors.Is(err, errFailVerifier) {
			t.Errorf("unexpected error from verification failure: %s", err)
		}
	}

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	select {
	case finalConf := <-d.Events():
		assert.Equal(t, "foo", finalConf.Foo)
		assert.Equal(t, "foo", d.View().Foo)
	case err := <-errCh:
		t.Errorf("unexpected error from monitor: %s", err)
	}
}

func TestWatcherWithDoneAndErrorCallback(t *testing.T) {
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

	reportedErrCh := make(chan error)
	p := Params[testConfig]{
		OnWatchedError: func(ctx context.Context, err error, oldConfig, newConfig *testConfig) {
			assert.Nil(t, newConfig)
			assert.NotNil(t, oldConfig)
			reportedErrCh <- err
		},
	}
	w := fakeWatchingSource{fakeSource: fakeSource{outVal: foozleConfig}}
	d, err := p.Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
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
	assert.Equal(t, "fim", c.Foo)
	assert.Equal(t, "fim", d.View().Foo)

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-d.Events()
	assert.Equal(t, "foo", finalConf.Foo)
	assert.Equal(t, "foo", d.View().Foo)

	repErr := errors.New("fizzlebizzle")
	assert.NoError(t, w.args.ReportError(ctx, repErr))

	receiveErr := <-reportedErrCh
	assert.Truef(t, errors.Is(receiveErr, repErr), "received %s", receiveErr)

	w.args.Done(ctx)
	runtime.Gosched()

	// after this point, reporting an error should either panic, block indefinitely or do nothing
	toCtx, toCancel := context.WithTimeout(ctx, time.Microsecond)
	defer toCancel()
	assert.Error(t, w.args.ReportError(toCtx, repErr))

	select {
	case w.args.(*watchArgs).c <- nil:
		t.Errorf("watchargs channel had successful send after Done() call")
	default:
	}
	select {
	case e := <-reportedErrCh:
		t.Errorf("unexpected call to error callback after shutdown: %s", e)
	default:
	}
}

func TestConfigWithNewConfigCallback(t *testing.T) {
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

	oldConf := make(chan *testConfig, 1)
	newConf := make(chan *testConfig)
	w := fakeWatchingSource{fakeSource: fakeSource{outVal: foozleConfig}}
	p := Params[testConfig]{
		OnWatchedError:          nil,
		SkipInitialVerification: false,
		OnNewConfig: func(ctx context.Context, oldConfig, newConfig *testConfig) {
			oldConf <- oldConfig
			newConf <- newConfig
		},
	}
	d, err := p.Config(ctx, &base, &fakeSource{outVal: emptyConf}, &w)
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
	c := <-newConf
	assert.Equal(t, "fim", c.Foo)
	oc := <-oldConf
	assert.Equal(t, "foozle", oc.Foo)
	assert.Equal(t, "fim", d.View().Foo)

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-newConf
	assert.Equal(t, "foo", finalConf.Foo)
	ocFinal := <-oldConf
	assert.Equal(t, "fim", ocFinal.Foo)
	assert.Equal(t, "foo", d.View().Foo)
}
