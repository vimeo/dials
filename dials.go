package dials

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/vimeo/dials/ptrify"
)

// WatchedErrorHandler is a callback that's called when something fails when
// dials is operating in a watching mode.  If non-nil, both oldConfig and
// newConfig are guaranteed to be populated with the same pointer-type that was
// passed to [Config]().
// newConfig will be nil for errors that prevent stacking.
type WatchedErrorHandler[T any] func(ctx context.Context, err error, oldConfig, newConfig *T)

// NewConfigHandler is a callback that's called after a new config is installed.
// Callbacks are run on a dedicated Goroutine, so one can do expensive/blocking
// work in this callback, however, execution should not last longer than the
// interval between new configs.
type NewConfigHandler[T any] func(ctx context.Context, oldConfig, newConfig *T)

// Params provides options for setting Dials's behavior in some cases.
type Params[T any] struct {
	// OnWatchedError is called when either of several conditions are met:
	//  - There is an error re-stacking the configuration
	//  - One of the Sources implementing the Watcher interface reports an error
	//  - a Verify() method fails after re-stacking when a new version is
	//    provided by a watching source
	OnWatchedError WatchedErrorHandler[T]

	// SkipInitialVerification skips the initial call to `Verify()` on any
	// configurations that implement the [VerifiedConfig] interface.
	//
	// In cases where later updates from Watching sources are depended upon to
	// provide a configuration that will be allowed by Verify(), one should set
	// this to true.  See `sourcewrap.Blank` for more details.
	//
	// Unlike DelayInitialVerification, this field only skips the initial Verify()
	// call, so all watching sources (including Blank) trigger configuration
	// verification.
	SkipInitialVerification bool

	// OnNewConfig is called when a new (valid) configuration is installed.
	//
	// OnNewConfig runs on the same "callback" goroutine as the
	// OnWatchedError callback, with callbacks being executed in-order.
	// In the event that a call to OnNewConfig blocks too long, some calls
	// may be dropped.
	OnNewConfig NewConfigHandler[T]

	// DelayInitialVerification skips calls to Verify() until the EnableVerification()
	// method is called.
	//
	// Some systems require coalescing the data from multiple Sources, which require
	// initialization parameters from other sources (e.g. filenames).
	//
	// Notably, many use-cases involving sourcewrap.Blank may require multiple steps
	// to initialize, during which time the configuration will be incomplete and may
	// not validate.
	DelayInitialVerification bool

	// CallGlobalCallbacksAfterVerificationEnabled suppresses calling any registered
	// global while in the delayed-verification mode.
	//
	// In particular, global callbacks (those registered in this struct) are
	// suppressed under two conditions:
	//  - DelayInitialVerification was set to true when Config was called
	//  - EnableVerification has not been called (without it returning an error)
	CallGlobalCallbacksAfterVerificationEnabled bool
}

// Config populates the passed in config struct by reading the values from the
// different Sources. The order of the sources denotes the precedence of the formats
// so the last source passed to the function has the ability to override fields that
// were set by previous sources
//
// If present, a Verify() method will be called after each stacking attempt.
// Blocking/expensive work should not be done in this method. (see the comment
// on Verify()) in [VerifiedConfig] for details)
//
// If complicated/blocking initialization/verification is necessary, one can either:
//   - If not using any watching sources, do any verification with the returned
//     config from Config.
//   - If using at least one watching source, configure a goroutine to watch the
//     channel returned by the `Dials.Events()` method that does its own
//     installation after verifying the config.
//
// More complicated verification/initialization should be done by
// consuming from the channel returned by `Events()`.
func (p Params[T]) Config(ctx context.Context, t *T, sources ...Source) (*Dials[T], error) {

	watcherChan := make(chan watchStatusUpdate)
	computed := make([]sourceValue, len(sources))

	typeOfT := reflect.TypeOf(t)
	if typeOfT.Kind() != reflect.Pointer {
		return nil, fmt.Errorf("config type %T is not a pointer", t)
	}

	tVal := realDeepCopy(t)

	valueCtx, cancelValues := context.WithCancel(ctx)
	defer cancelValues()

	typeInstance := &Type{ptrify.Pointerify(typeOfT.Elem(), tVal.Elem())}
	someoneWatching := false
	for i, source := range sources {
		s := source

		v, err := source.Value(valueCtx, typeInstance)
		if err != nil {
			return nil, err
		}
		computed[i] = sourceValue{
			source:   s,
			value:    v,
			watching: false,
		}

		if w, ok := source.(Watcher); ok {
			someoneWatching = true
			computed[i].watching = true
			wa := watchArgs{c: watcherChan, s: source}
			err = w.Watch(ctx, typeInstance, &wa)
			if err != nil {
				return nil, err
			}
		}
	}

	newValue, err := compose(tVal.Interface(), computed)
	if err != nil {
		return nil, err
	}

	nv, _ := newValue.(*T)

	d := &Dials[T]{
		updatesChan: make(chan *T, 1),
		params:      p,
	}
	d.value.Store(&versionedConfig[T]{serial: 0, cfg: nv})

	// Verify that the configuration is valid if a Verify() method is present.
	if vf, ok := newValue.(VerifiedConfig); ok && !p.SkipInitialVerification && !p.DelayInitialVerification {
		if vfErr := vf.Verify(); vfErr != nil {
			return nil, fmt.Errorf("initial configuration verification failed: %w", vfErr)
		}
	}

	// After this point, computed is owned by the monitor goroutine
	if someoneWatching {
		// Give the callback channel enough capacity that we
		// don't have to worry about dropping anything most of
		// the time.
		cbch := make(chan userCallbackEvent, 64)
		d.cbch = cbch
		cbmgr := callbackMgr[T]{
			p:  &p,
			ch: cbch,
		}
		go cbmgr.runCBs(ctx)

		monCtl := make(chan verifyEnable[T], 3)
		d.monCtl = monCtl
		go d.monitor(ctx, tVal.Interface().(*T), computed, watcherChan, monCtl)
	}
	return d, nil
}

// Config populates the passed in config struct by reading the values from the
// different Sources. The order of the sources denotes the precedence of the formats
// so the last source passed to the function has the ability to override fields that
// were set by previous sources
// This top-level function is present for convenience and backwards
// compatibility when there is no need to specify an error-handler.
func Config[T any](ctx context.Context, t *T, sources ...Source) (*Dials[T], error) {
	return Params[T]{}.Config(ctx, t, sources...)
}

// Source interface is implemented by each configuration source that is used to
// populate the config struct such as environment variables, command line flags,
// config files, and more
type Source interface {
	// Value provides the current value for the configuration.
	// Value methods should not create any long-lived resources or spin off
	// long-lived goroutines.
	// Config() will cancel the context passed to this method upon Config's
	// return.
	// Implementations that need to handle state changes with long-lived
	// background goroutines should implement the Watcher interface, which
	// explicitly provides a way to supply state updates.
	Value(context.Context, *Type) (reflect.Value, error)
}

// Decoder interface is implemented by different data formats to read the config
// files, decode the data, and insert the values in the config struct. Dials
// currently includes implementations for YAML, JSON, and TOML data formats.
type Decoder interface {
	Decode(io.Reader, *Type) (reflect.Value, error)
}

type valueUpdate struct {
	source    Source
	value     reflect.Value
	installed chan<- error
}

func (valueUpdate) isStatusReport() {}

type watcherDone struct {
	source Source
}

func (watcherDone) isStatusReport() {}

type watchErrorReport struct {
	source Source
	err    error
}

func (w *watchErrorReport) isStatusReport() {}

type watchStatusUpdate interface {
	isStatusReport()
}

type watchArgs struct {
	s Source
	c chan watchStatusUpdate
}

// ReportNewValue reports a new value. Returns an error if the internal
// reporting channel is full and the context expires/is-canceled.
func (w *watchArgs) ReportNewValue(ctx context.Context, val reflect.Value) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.c <- &valueUpdate{source: w.s, value: val}:
		return nil
	}
}

// BlockingReportNewValue reports a new value. Returns an error if the internal
// reporting channel is full and the context expires/is-canceled.
// Blocks until the new value has been or returns an error.
//
// Most Source implementations should use ReportNewValue(). This was added to
// support [github.com/vimeo/dials/sourcewrap.Blank]. This should only be used
// in similar cases.
func (w *watchArgs) BlockingReportNewValue(ctx context.Context, val reflect.Value) error {
	installed := make(chan error, 1)
	vu := valueUpdate{source: w.s, value: val, installed: installed}
	select {
	case <-ctx.Done():
		return fmt.Errorf("context expired while attempting to submit new value: %w", ctx.Err())
	case w.c <- &vu:
	}

	// Submitted, now we wait for the new value to be handled.
	select {
	case err := <-installed:
		if err != nil {
			return fmt.Errorf("stacking failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context expired while awaiting restack: %w", ctx.Err())
	}
}

// Done indicates that this watcher has stopped and will not send any
// more updates.
func (w *watchArgs) Done(ctx context.Context) {
	select {
	case <-ctx.Done():
	case w.c <- &watcherDone{source: w.s}:
	}
}

// ReportError reports a problem in the watcher. Returns an error if
// the internal reporting channel is full and the context
// expires/is-canceled.
func (w *watchArgs) ReportError(ctx context.Context, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.c <- &watchErrorReport{source: w.s, err: err}:
		return nil
	}
}

var _ WatchArgs = (*watchArgs)(nil)

// WatchArgs provides methods for a Watcher implementation to update the state
// of a Dials instance.
type WatchArgs interface {
	// ReportNewValue reports a new value. The base implementation returns an
	// error if the internal reporting channel is full and the context
	// expires/is-canceled, however, wrapping implementations are free to
	// return any other error as appropriate.
	ReportNewValue(ctx context.Context, val reflect.Value) error
	// Done indicates that this watcher has stopped and will not send any
	// more updates.
	Done(ctx context.Context)
	// ReportError reports a problem in the watcher. Returns an error if
	// the internal reporting channel is full and the context
	// expires/is-canceled.
	ReportError(ctx context.Context, err error) error

	// BlockingReportNewValue reports a new value. Returns an error if the internal
	// reporting channel is full and the context expires/is-canceled.
	// Blocks until the new value has been or returns an error.
	//
	// Most Source implementations should use ReportNewValue(). This was added to
	// support [github.com/vimeo/dials/sourcewrap.Blank]. This should only be used
	// in similar cases.
	BlockingReportNewValue(ctx context.Context, val reflect.Value) error
}

// Watcher should be implemented by Sources that allow their configuration to be
// watched for changes.
type Watcher interface {
	// Watch will be called in the primary goroutine calling Config(). If
	// Watcher implementations need a persistent goroutine, they should
	// spawn it themselves.
	Watch(context.Context, *Type, WatchArgs) error
}

// VerifiedConfig implements the Verify method, allowing Dials to execute the
// Verify method before returning/installing a new version of the
// configuration.
type VerifiedConfig interface {
	// Verify() should return a non-nil error if the configuration is
	// invalid.
	// As this method is called any time the configuration sources are
	// restacked, it should not do any complex or blocking work.
	Verify() error
}

// versionedConfig is the value-type of the value struct
type versionedConfig[T any] struct {
	serial uint64
	cfg    *T
}

// CfgSerial is an opaque object unique to a config-version
type CfgSerial[T any] struct {
	s   uint64
	cfg *T
}

// Events returns a channel that will get a message every time the configuration
// is updated.
//
// NOTE: In general, it is preferable to register a callback with
// [Dials.RegisterCallback], due to a cleaner interface and the ability to
// register multiple callbacks. Additionally, [NewConfigHandler]
// implementations get both the old and new configs, reducing the amount of
// state required to handle new events.
func (d *Dials[T]) Events() <-chan *T {
	return d.updatesChan
}

// Fill populates the passed struct with the current value of the configuration.
// It is a thin wrapper around assignment
//
// Deprecated: assign return value from View() instead. (this is a legacy method that predates generics)
func (d *Dials[T]) Fill(blankConfig *T) {
	*blankConfig = *d.View()
}

type userCallbackUnregisterToken[T any] struct {
	d *Dials[T]
	h *userCallbackHandle[T]
}

func (u *userCallbackUnregisterToken[T]) unregister(ctx context.Context) bool {
	doneCh := make(chan struct{})
	submitted := u.d.submitEventBlocking(ctx, &userCallbackUnregister[T]{
		handle: u.h,
		done:   doneCh,
	})
	if !submitted {
		return false
	}

	// Wait for the unregister "event" to be processed.
	select {
	case <-ctx.Done():
		return false
	case <-doneCh:
		return true
	}
}

// UnregisterCBFunc unregisters a callback from the dials object it was registered with.
type UnregisterCBFunc func(ctx context.Context) bool

// RegisterCallback registers the callback cb to receive notifications whenever
// a new configuration is installed. If the "current" version is later than the
// one represented by the value of [CfgSerial], a notification will be delivered immediately.
// This call is only blocking if the callback handling has filled up an
// internal channel. (likely because an already-registered callback is slow or
// blocking)
// serial must be obtained from [Dials.ViewVersion](). Catch-up callbacks are
// suppressed if passed passed an invalid CfgSerial (including the zero-value)
//
// Just like global callbacks, [NewConfigHandler] implementations execute serially on a single goroutine. As a result:
//   - callbacks will see config versions in the order they're installed
//   - the callbacks _may_ block, but only for time intervals that are short compared to the
//     interval between updates.
//
// May return a nil [UnregisterCBFunc] if the context expires
//
// The returned [UnregisterCBFunc] will block until the relevant callback has
// been removed from the set of callbacks.
func (d *Dials[T]) RegisterCallback(ctx context.Context, serial CfgSerial[T], cb NewConfigHandler[T]) UnregisterCBFunc {
	handle := userCallbackHandle[T]{
		cb:        cb,
		minSerial: serial.s,
	}
	submitted := d.submitEventBlocking(ctx, &userCallbackRegistration[T]{
		handle: &handle,
		serial: &serial,
	})

	if !submitted {
		return nil
	}
	tok := userCallbackUnregisterToken[T]{
		d: d,
		h: &handle,
	}
	return tok.unregister
}

// returns the new value (if any)
func (d *Dials[T]) updateSourceValue(
	ctx context.Context,
	t *T,
	skipVerify bool,
	sourceValues []sourceValue,
	watchTab *valueUpdate,
) *T {
	for i, sv := range sourceValues {
		if watchTab.source == sv.source {
			sourceValues[i].value = watchTab.value
			break
		}
	}
	newInterface, stackErr := compose(t, sourceValues)
	if stackErr != nil {
		oldVal := d.View()
		newVal, _ := newInterface.(*T)
		d.submitEvent(ctx, &watchErrorEvent[T]{
			err: stackErr, oldConfig: oldVal, newConfig: newVal,
		})
		if watchTab.installed != nil {
			watchTab.installed <- stackErr
		}
		return nil
	}

	// Verify that the configuration is valid if a Verify() method is present.
	if vf, ok := newInterface.(VerifiedConfig); ok && !skipVerify {
		if vfErr := vf.Verify(); vfErr != nil {
			oldVal := d.View()

			newVal := newInterface.(*T)

			d.submitEvent(ctx, &watchErrorEvent[T]{
				err: vfErr, oldConfig: oldVal, newConfig: newVal,
			})

			if watchTab.installed != nil {
				watchTab.installed <- vfErr
			}
			return nil
		}
	}

	newVers := newInterface.(*T)

	_, oldSerial := d.ViewVersion()

	// We can do a blind-store here because this goroutine (monitor()) has
	// exclusive ownership of writes to this atomic-value
	d.value.Store(&versionedConfig[T]{serial: oldSerial.s + 1, cfg: newVers})
	select {
	case d.updatesChan <- newVers:
	default:
	}

	// If there's an installed channel, poke it.
	if watchTab.installed != nil {
		watchTab.installed <- nil
	}

	return newVers
}

func (d *Dials[T]) markSourceDone(
	ctx context.Context,
	sourceValues []sourceValue,
	watchTab *watcherDone,
) bool {
	// Set the calling source's watching bit to false
	for i, sv := range sourceValues {
		if watchTab.source == sv.source {
			sourceValues[i].watching = false
			break
		}
	}

	// check whether any sources have watching set to true
	// (using a loop here because it's not worth maintaining an extra
	// datastructure for an infrequent operation)
	for _, sv := range sourceValues {
		if sv.watching {
			return true
		}
	}
	return false
}

func (d *Dials[T]) submitEventBlocking(ctx context.Context, ev userCallbackEvent) bool {
	// don't panic
	if d.cbch == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case d.cbch <- ev:
		return true
	}
}

func (d *Dials[T]) submitEvent(ctx context.Context, ev userCallbackEvent) {
	// don't panic
	if d.cbch == nil {
		return
	}
	select {
	case <-ctx.Done():
	case d.cbch <- ev:
		// never block we'd rather drop callbacks than deadlock the watchers
	default:
	}
}

type verifyEnableResp[T any] struct {
	// only one of error or cfgTok will be returned
	err error
	v   *T
	tok CfgSerial[T]
}

// verifyEnable is the payload type for the monCtl channel used to signal that
// verification should be enabled.
type verifyEnable[T any] struct {
	// resp must have capacity 1
	resp chan<- verifyEnableResp[T]
}

// EnableVerification enables verification on dials if DelayInitialVerification was set on
// the [Params] struct. Returns the config that was verified and a [CfgSerial] or the
// error from calling Verify() (if the config type implements [VerifiedConfig].
//
// If DelayInitialVerification is not set, returns successfully without verifying the
// config.
//
// If verification succeeds on the currently installed configuration, all subsequent
// configuration versions will be verified. (based on re-stacking versions from watching
// sources)
//
// When there are watching sources (including Blank) the global callbacks may
// be suppressed with the [Params].CallGlobalCallbacksAfterVerificationEnabled option.
// This suppression expires after verification is re-enabled by this method.
//
// Note: if the context expires while this call is awaiting a response from the background
// "monitor" goroutine, verification may still happen, but whether it transitions out of
// the delayed verification state is indeterminate.
func (d *Dials[T]) EnableVerification(ctx context.Context) (*T, CfgSerial[T], error) {
	if !d.params.DelayInitialVerification {
		// this is a noop, since we never disabled verification
		cfg, tok := d.ViewVersion()
		return cfg, tok, nil
	} else if d.monCtl == nil {
		cfg, tok := d.ViewVersion()
		if vc, ok := any(cfg).(VerifiedConfig); ok {
			return nil, CfgSerial[T]{}, vc.Verify()
		}
		return cfg, tok, nil
	}
	// must have capacity 1
	resp := make(chan verifyEnableResp[T], 1)
	select {
	case d.monCtl <- verifyEnable[T]{resp: resp}:
	case <-ctx.Done():
		return nil, CfgSerial[T]{}, fmt.Errorf("context expired while signaling: %w", ctx.Err())
	}

	select {
	case r := <-resp:
		return r.v, r.tok, r.err
	case <-ctx.Done():
		return nil, CfgSerial[T]{}, fmt.Errorf("context expired while awaiting response: %w", ctx.Err())
	}

}

func (d *Dials[T]) monitorEnableVerify(ve verifyEnable[T]) bool {
	vt, serial := d.ViewVersion()
	if vf, ok := any(vt).(VerifiedConfig); ok {
		if vfErr := vf.Verify(); vfErr != nil {
			ve.resp <- verifyEnableResp[T]{
				err: vfErr,
				v:   nil,
				tok: CfgSerial[T]{},
			}

			return false
		}
	}
	ve.resp <- verifyEnableResp[T]{
		err: nil,
		v:   vt,
		tok: serial,
	}
	return true
}

func (d *Dials[T]) monitor(
	ctx context.Context,
	t *T,
	sourceValues []sourceValue,
	watcherChan chan watchStatusUpdate,
	monCtl <-chan verifyEnable[T],
) {
	defer close(d.cbch)
	skipVerify := d.params.DelayInitialVerification
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-monCtl:
			if !skipVerify {
				// we're not in skipVerify mode, so just send back
				// a success and continue
				cfg, serial := d.ViewVersion()
				v.resp <- verifyEnableResp[T]{
					err: nil,
					v:   cfg,
					tok: serial,
				}
				continue
			}
			skipVerify = !d.monitorEnableVerify(v)
		case watchTab := <-watcherChan:
			switch v := watchTab.(type) {
			case *valueUpdate:
				oldConfig, oldSerial := d.ViewVersion()
				newConfig := d.updateSourceValue(ctx, t, skipVerify, sourceValues, v)
				if newConfig != nil {
					d.submitEvent(ctx, &newConfigEvent[T]{
						oldConfig: oldConfig,
						newConfig: newConfig,
						serial:    oldSerial.s + 1,
						globalCBsSuppressed: skipVerify &&
							d.params.CallGlobalCallbacksAfterVerificationEnabled,
					})
				}
			case *watchErrorReport:
				if !skipVerify && !d.params.CallGlobalCallbacksAfterVerificationEnabled {
					d.submitEvent(ctx, &watchErrorEvent[T]{
						err: fmt.Errorf("error reported by source of type %T: %w",
							v.source, v.err),
						oldConfig: d.View(),
						newConfig: nil,
					})
				}
			case *watcherDone:
				if !d.markSourceDone(ctx, sourceValues, v) {
					// if there are no watching sources, just exit.
					return
				}
			default:
				panic(fmt.Errorf("unexpected type %[1]T: %+[1]v", watchTab))
			}
		}
	}
}

func compose(t any, sources []sourceValue) (any, error) {
	copyValuePtr := realDeepCopy(t)
	value := copyValuePtr.Elem()
	for _, source := range sources {
		// automatically dereference pointers that may be in the value
		s := source.value
		if s.Kind() == reflect.Pointer {
			s = s.Elem()
		}
		o := newOverlayer()
		sv := o.dc.deepCopyValue(s)
		if overlayErr := o.overlayStruct(value, sv); overlayErr != nil {
			return nil, overlayErr
		}

	}

	return value.Addr().Interface(), nil
}

type sourceValue struct {
	source   Source
	value    reflect.Value
	watching bool
}

// Type is a wrapper for a reflect.Type.
type Type struct {
	t reflect.Type
}

// Type describes a config struct type, usually it is already pointerified
func (t *Type) Type() reflect.Type {
	return t.t
}

// NewType constructs a new dials Type for a reflect.Type.
func NewType(t reflect.Type) *Type {
	return &Type{
		t: t,
	}
}
