package dials

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
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

func TestConfigWithNewConfigCallbacks(t *testing.T) {
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
	_, initSerial := d.ViewVersion()

	// no catch-up for either of these. We've registered the config with
	// the only version of the config that's ever existed (for this dials object).
	// we'll unregister it immediately.
	assert.True(t, d.RegisterCallback(ctx, initSerial, func(ctx context.Context, oldConf, newConf *testConfig) {
		t.Error("unexpected call of config with current version")
	})(ctx), "unregister failed")
	// similar, but using a zero-value for the serial (again, unregister immediately
	assert.True(t, d.RegisterCallback(ctx, CfgSerial[testConfig]{}, func(ctx context.Context, oldConf, newConf *testConfig) {
		t.Error("unexpected call of config with zero-valued serial")
	})(ctx), "unregister failed")

	// We'll be setting this value on the next config (just a bit below).
	fimStr := "fim"

	// we'll register 30 callbacks, and then unregister them after the next event
	cbcalls := uint32(0)
	unregCBs := make([]UnregisterCBFunc, 30)
	for z := range unregCBs {
		idx := z
		CBcallCount := 0
		cb := func(ctx context.Context, oldConf, newConf *testConfig) {
			atomic.AddUint32(&cbcalls, 1)
			assert.Equalf(t, newConf.Foo, fimStr, "cb %d", idx)
			CBcallCount++
			if CBcallCount > 1 {
				t.Errorf("callback %d called too many times: %d", idx, CBcallCount)
			}
		}
		unregCBs[z] = d.RegisterCallback(ctx, initSerial, cb)
	}

	// Push a new value, that should overlay on top of the base
	fimConfig := ptrifiedConfig{
		Foo: &fimStr,
	}
	w.send(ctx, reflect.ValueOf(fimConfig))
	c := <-newConf
	assert.Equal(t, "fim", c.Foo)
	oc := <-oldConf
	assert.Equal(t, "foozle", oc.Foo)
	assert.Equal(t, "fim", d.View().Foo)

	// unregister all the callbacks we registered above.
	for _, unregCB := range unregCBs {
		assert.True(t, unregCB(ctx))
	}

	// we're now certain that all our callbacks have been called and
	// unregistered. Check the value of cbcalls.
	assert.Equal(t, uint32(len(unregCBs)), atomic.LoadUint32(&cbcalls))

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-newConf
	assert.Equal(t, "foo", finalConf.Foo)
	ocFinal := <-oldConf
	assert.Equal(t, "fim", ocFinal.Foo)
	assert.Equal(t, "foo", d.View().Foo)
}

func TestConfigWithNewConfigCallbacksSaturate(t *testing.T) {
	t.Parallel()
	type testConfig struct {
		Foo string
		Gen uint64
	}

	type ptrifiedConfig struct {
		Foo *string
		Gen *uint64
	}

	base := testConfig{
		Foo: "foo",
		Gen: 0,
	}
	emptyConf := ptrifiedConfig{
		Foo: nil,
		Gen: nil,
	}
	uint64Ptr := func(z uint64) *uint64 {
		return &z
	}
	// Push a new value, that should overlay on top of the base
	foozleStr := "foozle"
	foozleConfig := ptrifiedConfig{
		Foo: &foozleStr,
		Gen: uint64Ptr(0),
	}

	// setup a cancelable context so the monitor goroutine gets shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// give oldConf a large capacity (we don't want to block on both)
	oldConf := make(chan *testConfig, 128)
	// we'll use this to block up the callback goroutine
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
	_, initSerial := d.ViewVersion()

	// we don't actually want to fill up the event channel, but give enough configs
	cfgCnt := cap(d.cbch) - 20

	// we'll register 30 callbacks, and then unregister them after the next event
	cbcalls := uint32(0)
	unregCBs := make([]UnregisterCBFunc, 30)
	for z := range unregCBs {
		idx := z
		CBcallCount := 0
		cb := func(ctx context.Context, oldConf, newConf *testConfig) {
			atomic.AddUint32(&cbcalls, 1)
			expFooVal := "fizzle" + strconv.Itoa(CBcallCount)
			if CBcallCount == cfgCnt {
				expFooVal = "fim"
			}
			assert.Equalf(t, newConf.Foo, expFooVal, "cb %d", idx)
			CBcallCount++
			if CBcallCount > cfgCnt+1 {
				t.Errorf("callback %d called too many times: %d", idx, CBcallCount)
			}
		}
		unregCBs[z] = d.RegisterCallback(ctx, initSerial, cb)
	}

	// Before we send new events, make sure that everything's caught up
	// (the unregister call waits until its message is processed)
	assert.True(t, d.RegisterCallback(ctx, CfgSerial[testConfig]{}, func(ctx context.Context, oldConf, newConf *testConfig) {
		t.Error("unexpected call of config with zero-valued serial")
	})(ctx), "unregister failed")

	for z := 0; z < cfgCnt; z++ {
		expFooVal := "fizzle" + strconv.Itoa(z)
		fimConfig := ptrifiedConfig{
			Foo: &expFooVal,
			Gen: uint64Ptr(uint64(z) + 1),
		}
		w.send(ctx, reflect.ValueOf(fimConfig))
	}
	// We now have a channel full of deferred callbacks, which will get
	// delivered (since we filled up the channel).
	fullVers, fullSerial := d.ViewVersion()
	if !strings.HasPrefix(fullVers.Foo, "fizzle") {
		t.Errorf("unexpected fullVers.Foo prefix: %s", fullVers.Foo)
	}
	if fullVers.Gen > uint64(cfgCnt) {
		t.Errorf("Gen value too large: %d", fullVers.Gen)
	}

	regUnregDone := make(chan struct{})

	// this will block until we've cleared out the newConf values.
	go func() {
		assert.True(t, d.RegisterCallback(ctx, fullSerial, func(ctx context.Context, oldConf, newConf *testConfig) {
			// we have a race in registration, skip anything with a
			// generation between the one on fullVers and the last
			// enqueued value so far
			if newConf.Gen > fullVers.Gen && newConf.Gen <= uint64(cfgCnt) {
				return
			}
			t.Errorf("unexpected call of config with later version (got Gen: %d)", newConf.Gen)
		})(ctx), "unregister failed")
		close(regUnregDone)
	}()

	for z := 0; z < cfgCnt; z++ {
		c := <-newConf
		expFooVal := "fizzle" + strconv.Itoa(int(c.Gen-1))
		assert.Equal(t, expFooVal, c.Foo)
		<-oldConf
	}
	// now, our unregister should have completed
	<-regUnregDone

	// We'll be setting this value on the next config (just a bit below).
	fimStr := "fim"
	// Push a new value, that should overlay on top of the base
	fimConfig := ptrifiedConfig{
		Foo: &fimStr,
		Gen: uint64Ptr(uint64(cfgCnt) + 2),
	}
	w.send(ctx, reflect.ValueOf(fimConfig))
	c := <-newConf
	assert.Equal(t, "fim", c.Foo)
	assert.Equal(t, uint64(cfgCnt)+2, c.Gen)
	oc := <-oldConf
	assert.Equal(t, "fizzle43", oc.Foo)
	assert.Equal(t, uint64(cfgCnt), oc.Gen)
	assert.Equal(t, "fim", d.View().Foo)

	// unregister all the callbacks we registered above.
	for _, unregCB := range unregCBs {
		assert.True(t, unregCB(ctx))
	}

	// we're now certain that all our callbacks have been called and
	// unregistered. Check the value of cbcalls.
	assert.Equal(t, uint32(len(unregCBs)*(cfgCnt+1)), atomic.LoadUint32(&cbcalls))

	// push another empty config
	w.send(ctx, reflect.ValueOf(emptyConf))
	finalConf := <-newConf
	assert.Equal(t, "foo", finalConf.Foo)
	ocFinal := <-oldConf
	assert.Equal(t, "fim", ocFinal.Foo)
	assert.Equal(t, "foo", d.View().Foo)
}

func ExampleDials_RegisterCallback() {
	// setup a cancelable context so the monitor goroutine gets shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testConfig struct {
		Foo string
	}

	type ptrifiedConfig struct {
		Foo *string
	}

	base := testConfig{
		Foo: "foo",
	}
	// Push a new value, that should overlay on top of the base
	foozleStr := "foozle"
	foozleConfig := ptrifiedConfig{
		Foo: &foozleStr,
	}

	w := fakeWatchingSource{fakeSource: fakeSource{outVal: foozleConfig}}
	d, dialsErr := Config(ctx, &base, &w)
	if dialsErr != nil {
		panic("unexpected error: " + dialsErr.Error())
	}

	cfg, serialToken := d.ViewVersion()
	fmt.Printf("Foo: %s\n", cfg.Foo)

	unregCB := d.RegisterCallback(ctx, serialToken, func(ctx context.Context, oldCfg, newCfg *testConfig) {
		panic("not getting called this time")
	})

	unregCB(ctx)

	// Output:
	// Foo: foozle
}
