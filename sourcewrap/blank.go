package sourcewrap

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/vimeo/dials"
)

// Blank operates as a blank Source in its default state. It provides a
// SetSource method for updating the inner Source later and uses the
// dials.Watcher interface to update the View it's inserted into.
//
// Blanks cannot be reused as they have to be aware of parameters of a
// particular View.
//
// Note that when using this, consider setting `SkipInitialVerification` on
// `dials.Params` to `true` if any data from Blank-wrapped sources is considered
// critical by the `Verify()` method provided by a configuration implementing
// `dials.VerifiedConfig`.
//
// For instance, consider when one's configuration implements
// `dials.VerifiedConfig` and should receive data from two Sources, one being
// wrapped by `sourcewrap.Blank`. When `SkipInitialVerification` is `false`,
// `params.Config` will call `Verify()` before the Blank-wrapped source has a
// chance to have its inner Source updated. Therefore, this initial call to
// `Verify()` will only have data provided by the non-Blank-wrapped source.
type Blank struct {
	inner dials.Source
	mu    sync.Mutex
	// arguments to Watch() so we can use them later, and pass them on if the
	// new Source is a Watcher.
	watchCtx context.Context
	wa       dials.WatchArgs
	t        *dials.Type
}

var _ dials.Source = (*Blank)(nil)
var _ dials.Watcher = (*Blank)(nil)

func (b *Blank) getInner() dials.Source {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.inner
}

// Value implements the dials.Source interface, returning either a zero-value
// for the type it's passed, or delegating to the wrapped source (if present).
func (b *Blank) Value(ctx context.Context, t *dials.Type) (reflect.Value, error) {
	inner := b.getInner()
	if inner != nil {
		return inner.Value(ctx, t)
	}
	return reflect.New(t.Type()), nil
}

// Watch implements the dials.Watcher interface.
// It is necessary so calls to SetSource can notify the containing View of a new value.
func (b *Blank) Watch(ctx context.Context,
	t *dials.Type, wa dials.WatchArgs) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.t != nil {
		return fmt.Errorf("blank has already been used, with type %s", b.t.Type())
	}
	b.t = t
	b.wa = wa
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
//
// It is safe to call SetSource multiple times with different Sources, however,
// once a Watcher is set as the inner source, it takes over ownership of the
// state for the slot and cannot be replaced.
func (b *Blank) SetSource(ctx context.Context, s dials.Source) error {
	if s == nil {
		return fmt.Errorf("cannot pass a nil source to *Blank.SetSource with type %s",
			b.t.Type())
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.inner != nil {
		if _, isWatcher := b.inner.(dials.Watcher); isWatcher {
			return fmt.Errorf("disallowed attempt to replace Watcher Source: %T",
				b.inner)
		}
	}

	v, err := s.Value(ctx, b.t)
	if err != nil {
		return &wrappedErr{prefix: "initial call to Value failed: ", err: err}
	}
	b.inner = s
	if newValErr := b.wa.ReportNewValue(ctx, v); newValErr != nil {
		return fmt.Errorf("failed to propagate change: %w", newValErr)
	}

	if w, ok := s.(dials.Watcher); ok {
		wErr := w.Watch(b.watchCtx, b.t, b.wa)
		if wErr != nil {
			return &wrappedErr{prefix: "call to Watch failed: ", err: wErr}
		}
	}
	return nil
}

// Done instructs Dials that this Blank source will never be used in a watching
// mode ever again (allowing Dials to shutdown a goroutine once all other
// sources implementing Watcher have called Done()).
// This call is a nop if a source implementing the Watcher interface is present
// within Blank. (Semantically, the WatchArgs are no longer the Blank's once a
// Watcher-implementing Source is set)
func (b *Blank) Done(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.inner.(type) {
	case dials.Watcher:
		return
	default:
	}
	if b.wa == nil {
		return
	}
	b.wa.Done(ctx)
}
