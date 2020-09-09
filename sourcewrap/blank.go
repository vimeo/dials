package sourcewrap

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/vimeo/dials"
)

// Blank operates as a blank source in its default state, but has a SetSource
// method for setting the source later, it then uses the dials.Watcher
// interface to update the View it's inserted into.
//
// Blanks cannot be reused as they have to be aware of parameters of a
// particular View.
type Blank struct {
	inner dials.Source
	mu    sync.Mutex
	// arguments to Watch() so we can use them later, and pass them on if the
	// new Source is a Watcher.
	watchCtx context.Context
	cb       func(context.Context, reflect.Value)
	t        *dials.Type
}

func (b *Blank) getInner() dials.Source {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.inner
}

// Value implements the dials.Source interface, returning either a zero-value
// for the type it's passed, or delegating to the wrapped source (if present).
func (b *Blank) Value(t *dials.Type) (reflect.Value, error) {
	inner := b.getInner()
	if inner != nil {
		return inner.Value(t)
	}
	return reflect.New(t.Type()), nil
}

// Watch implements the dials.Watcher interface.
// It is necessary so calls to SetSource can notify the containing View of a new value.
func (b *Blank) Watch(ctx context.Context,
	t *dials.Type, cb func(context.Context, reflect.Value)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.t != nil {
		return fmt.Errorf("blank has already been used, with type %s", b.t.Type())
	}
	b.t = t
	b.cb = cb
	b.watchCtx = ctx
	return nil
}

type wrappedErr struct {
	prefix string
	err    error
}

func (w *wrappedErr) Error() string {
	return w.prefix + w.err.Error()
}

func (w *wrappedErr) Unwrap() error {
	return w.err
}

// SetSource sets the wrapped source, switching the Blank into full delegation mode.
// The new Source's Value() method will be called, and the return value will be
// pushed to the view via the asynchronous watch interface.
// If the new Source implements dials.Watcher, its Watch method will be called.
func (b *Blank) SetSource(ctx context.Context, s dials.Source) error {
	if s == nil {
		return fmt.Errorf("cannot pass a nil source to *Blank.SetSource with type %s",
			b.t.Type())
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	// TODO: this seems wrong, Value should take a context as well.
	v, err := s.Value(b.t)
	if err != nil {
		return &wrappedErr{prefix: "initial call to Value failed: ", err: err}
	}
	b.inner = s
	b.cb(ctx, v)

	if w, ok := s.(dials.Watcher); ok {
		wErr := w.Watch(b.watchCtx, b.t, b.cb)
		if wErr != nil {
			return &wrappedErr{prefix: "call to Watch failed: ", err: wErr}
		}
	}
	return nil
}
