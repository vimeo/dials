package dials

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync/atomic"

	"github.com/vimeo/dials/ptrify"
)

// WatchedErrorHandler is a callback that's called when something fails when
// dials is operating in a watching mode.  Both oldConfig and NewConfig are
// guaranteed to be populated with the same pointer-type that was passed to
// `Config()`.
type WatchedErrorHandler func(ctx context.Context, err error, oldConfig, newConfig interface{})

// Params provides options for setting Dials's behavior in some cases.
type Params struct {
	// OnWatchedError is called when either of several conditions are met:
	//  - There is an error re-stacking the configuration
	//  -
	//  - a Verify() method fails after re-stacking when a new version is
	//    provided by a watching source
	OnWatchedError WatchedErrorHandler

	// SkipInitialVerification doesn't run the initial verification that would
	// generally happen while calling `Config`.  This is useful when there are
	// `Blank` sources that have dependent values in other sources and we don't
	// expect to have a fully valid config until the Blank is appropriately
	// replaced.  See the `ez` package.
	SkipInitialVerification bool
}

// Config populates the passed in config struct by reading the values from the
// different Sources. The order of the sources denotes the precedence of the formats
// so the last source passed to the function has the ability to override fields that
// were set by previous sources
//
// If present, a Verify() method will be called after each stacking attempt.
// Blocking/expensive work should not be done in this method. (see the comment
// on Verify()) in VerifiedConfig for details)
//
// If complicated/blocking initialization/verification is necessary, one can either:
//  - If not using any watching sources, do any verification with the returned
//    config from Config.
//  - If using at least one watching source, configure a goroutine to watch the
//    channel returned by the `Dials.Events()` method that does its own
//    installation after verifying the config.
//
// More complicated verification/initialization should be done by
// consuming from the channel returned by `Events()`.
func (p Params) Config(ctx context.Context, t interface{}, sources ...Source) (*Dials, error) {

	watcherChan := make(chan watchStatusUpdate)
	computed := make([]sourceValue, len(sources))

	typeOfT := reflect.TypeOf(t)
	if typeOfT.Kind() != reflect.Ptr {
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

	d := &Dials{
		value:       atomic.Value{},
		updatesChan: make(chan interface{}, 1),
		params:      p,
	}
	d.value.Store(newValue)

	// Verify that the configuration is valid if a Verify() method is present.
	if vf, ok := newValue.(VerifiedConfig); ok && !p.SkipInitialVerification {
		if vfErr := vf.Verify(); vfErr != nil {
			return nil, fmt.Errorf("Initial configuration verification failed: %w", vfErr)
		}
	}

	// After this point, computed is owned by the monitor goroutine
	if someoneWatching {
		go d.monitor(ctx, tVal.Interface(), computed, watcherChan)
	}
	return d, nil
}

// Config populates the passed in config struct by reading the values from the
// different Sources. The order of the sources denotes the precedence of the formats
// so the last source passed to the function has the ability to override fields that
// were set by previous sources
// This top-level function is present for convenience and backwards
// compatibility when there is no need to specify an error-handler.
func Config(ctx context.Context, t interface{}, sources ...Source) (*Dials, error) {
	return Params{}.Config(ctx, t, sources...)
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
	source Source
	value  reflect.Value
}

func (valueUpdate) isStatusReport() {}

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

// WatchArgs provides methods for a Watcher implementation to update the state
// of a Dials instance.
type WatchArgs interface {
	// ReportNewValue reports a new value. The base implementation returns an
	// error if the internal reporting channel is full and the context
	// expires/is-canceled, however, wrapping implementations are free to
	// return any other error as appropriate.
	ReportNewValue(ctx context.Context, val reflect.Value) error
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

// Dials is the main access point for your configuration.
type Dials struct {
	value       atomic.Value
	updatesChan chan interface{}
	params      Params
}

// View returns the configuration struct populated.
func (d *Dials) View() interface{} {
	return d.value.Load()
}

// Events returns a channel that will get a message every time the configuration
// is updated.
func (d *Dials) Events() <-chan interface{} {
	return d.updatesChan
}

// Fill populates the passed struct with the current value of the configuration.
// It will panic if the type of `blankConfig` does not match the type of the
// configuration value passed to `Config` in the first place.
func (d *Dials) Fill(blankConfig interface{}) {
	bVal := reflect.ValueOf(blankConfig)
	currentVal := reflect.ValueOf(d.value.Load())

	if bVal.Type() != currentVal.Type() {
		panic(fmt.Sprintf(
			"value to fill type (%s) does not match actual type (%s)",
			bVal.Type(),
			currentVal.Type(),
		))
	}

	bVal.Elem().Set(currentVal.Elem())
}

func (d *Dials) updateSourceValue(
	ctx context.Context,
	t interface{},
	sourceValues []sourceValue,
	watchTab *valueUpdate,
) {
	for i, sv := range sourceValues {
		if watchTab.source == sv.source {
			sourceValues[i].value = watchTab.value
			break
		}
	}
	newInterface, stackErr := compose(t, sourceValues)
	if stackErr != nil {
		if d.params.OnWatchedError != nil {
			d.params.OnWatchedError(
				ctx, stackErr, d.value.Load(), newInterface)
		}
		return
	}

	// Verify that the configuration is valid if a Verify() method is present.
	if vf, ok := newInterface.(VerifiedConfig); ok {
		if vfErr := vf.Verify(); vfErr != nil {
			if d.params.OnWatchedError != nil {
				d.params.OnWatchedError(
					ctx, vfErr, d.value.Load(), newInterface)
			}
			return
		}
	}

	d.value.Store(newInterface)
	select {
	case d.updatesChan <- newInterface:
	default:
	}
}

func (d *Dials) monitor(
	ctx context.Context,
	t interface{},
	sourceValues []sourceValue,
	watcherChan chan watchStatusUpdate,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case watchTab := <-watcherChan:
			switch v := watchTab.(type) {
			case *valueUpdate:
				d.updateSourceValue(ctx, t, sourceValues, v)
			default:
				panic(fmt.Errorf("unexpected type %[1]T: %+[1]v", watchTab))
			}
		}
	}
}

func compose(t interface{}, sources []sourceValue) (interface{}, error) {
	copyValuePtr := realDeepCopy(t)
	value := copyValuePtr.Elem()
	for _, source := range sources {
		// automatically dereference pointers that may be in the value
		s := source.value
		if s.Kind() == reflect.Ptr {
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
