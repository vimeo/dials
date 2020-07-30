package dials

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync/atomic"

	"github.com/vimeo/dials/ptrify"
)

// Config ...
func Config(ctx context.Context, t interface{}, sources ...Source) (*Dials, error) {

	watcherChan := make(chan *watchTab)
	computed := make([]sourceValue, 0, len(sources))

	typeOfT := reflect.TypeOf(t)
	if typeOfT.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("config type %T is not a pointer", t)
	}

	tVal := realDeepCopy(t)

	typeInstance := &Type{ptrify.Pointerify(typeOfT.Elem(), tVal.Elem())}
	someoneWatching := false
	for _, source := range sources {
		s := source

		v, err := source.Value(typeInstance)
		if err != nil {
			return nil, err
		}
		computed = append(computed, sourceValue{
			source: s,
			value:  v,
		})

		if w, ok := source.(Watcher); ok {
			someoneWatching = true
			err = w.Watch(ctx, typeInstance, func(ctx context.Context, v reflect.Value) {
				select {
				case <-ctx.Done():
				case watcherChan <- &watchTab{source: s, value: v}:
				}
			})
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
	}
	d.value.Store(newValue)

	if someoneWatching {
		go d.monitor(ctx, tVal.Interface(), computed, watcherChan)
	}
	return d, nil
}

// Source ...
type Source interface {
	Value(*Type) (reflect.Value, error)
}

// Decoder ...
type Decoder interface {
	Decode(io.Reader, *Type) (reflect.Value, error)
}

type watchTab struct {
	source Source
	value  reflect.Value
}

// Watcher should be implemented by Sources that allow their configuration to be
// watched for changes.
type Watcher interface {
	Watch(context.Context, *Type, func(context.Context, reflect.Value)) error
}

// Dials is the main access point for your configuration.
type Dials struct {
	value       atomic.Value
	updatesChan chan interface{}
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

func (d *Dials) monitor(
	ctx context.Context,
	t interface{},
	sourceValues []sourceValue,
	watcherChan chan *watchTab,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case watchTab := <-watcherChan:
			for i, sv := range sourceValues {
				if watchTab.source == sv.source {
					sourceValues[i].value = watchTab.value
					break
				}
			}
			newInterface, err := compose(t, sourceValues)
			if err != nil {
				continue
			}
			d.value.Store(newInterface)
			select {
			case d.updatesChan <- newInterface:
			default:
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
	source Source
	value  reflect.Value
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
