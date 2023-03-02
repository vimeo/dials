package sourcewrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/vimeo/dials"
)

// returns an empty value, but increments a counter for every call to Value()
// (not thread-safe)
type trivalCountingSource struct {
	callCount uint32
}

func (t *trivalCountingSource) Value(_ context.Context, typ *dials.Type) (reflect.Value, error) {
	t.callCount++
	return reflect.New(typ.Type()), nil
}

// returns an error, but increments a counter for every call to Value()
// (not thread-safe)
type trivalErroringSource struct {
	callCount uint32
	err       error
}

func (t *trivalErroringSource) Value(_ context.Context, typ *dials.Type) (reflect.Value, error) {
	t.callCount++
	return reflect.Value{}, t.err
}

// returns an empty value, but increments a counter for every call to Value()
// (not thread-safe)
type trivalCountingWatchingSource struct {
	callCount   uint32
	watchcalled bool
	args        dials.WatchArgs
	typ         *dials.Type
}

var _ dials.Watcher = (*trivalCountingWatchingSource)(nil)

func (t *trivalCountingWatchingSource) Watch(ctx context.Context, typ *dials.Type, args dials.WatchArgs) error {
	t.watchcalled = true
	t.args = args
	t.typ = typ
	return nil
}

func (t *trivalCountingWatchingSource) Value(_ context.Context, typ *dials.Type) (reflect.Value, error) {
	t.callCount++
	return reflect.New(typ.Type()), nil
}

func (t *trivalCountingWatchingSource) poke(ctx context.Context) {
	t.args.ReportNewValue(ctx, reflect.New(t.typ.Type()))
}

type trivalErroringWatchingSource struct {
	callCount   uint32
	watchcalled bool
	err         error
}

var _ dials.Watcher = (*trivalErroringWatchingSource)(nil)

func (t *trivalErroringWatchingSource) Watch(ctx context.Context, typ *dials.Type, args dials.WatchArgs) error {
	t.watchcalled = true
	return t.err
}

func (t *trivalErroringWatchingSource) Value(_ context.Context, typ *dials.Type) (reflect.Value, error) {
	t.callCount++
	return reflect.New(typ.Type()), nil
}

func TestBlankSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	b := Blank{}
	type basicConf struct {
		A int
		B int
		C string
	}
	c := basicConf{
		A: 3,
		B: 4809,
		C: "fob",
	}
	d, err := dials.Config(ctx, &c, &b)
	if err != nil {
		t.Fatalf("failed to construct View: %s", err)
	}
	initConf := d.View()
	if *initConf != c {
		t.Errorf("unexpected initial config: got %+v; expected %+v", *initConf, c)
	}

	triv := trivalCountingSource{}

	setErr := b.SetSource(ctx, &triv)
	if setErr != nil {
		t.Errorf("b.SetSource() failed with trivial nop impl: %s", setErr)
	}

	// SetSource should have installed the new version.
	newConf := d.View()
	if *newConf != c {
		t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, c)
	}
	if triv.callCount != 1 {
		t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
	}
}

func TestBlankSourceError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	b := Blank{}
	type basicConf struct {
		A int
		B int
		C string
	}
	c := basicConf{
		A: 3,
		B: 4809,
		C: "fob",
	}
	d, err := dials.Config(ctx, &c, &b)
	if err != nil {
		t.Fatalf("failed to construct View: %s", err)
	}
	initConf := d.View()
	if *initConf != c {
		t.Errorf("unexpected initial config: got %+v; expected %+v", *initConf, c)
	}

	expErr := errors.New("foobarbaz")
	triv := trivalErroringSource{err: expErr}

	setErr := b.SetSource(ctx, &triv)
	if setErr == nil {
		t.Errorf("b.SetSource() did not fail as expected with erroring source")
	} else if wErr, ok := setErr.(*wrappedErr); !ok || wErr.Unwrap() != expErr {
		t.Errorf("b.SetSource() failed with with an unexpected error: %s", setErr)
	}

	if triv.callCount != 1 {
		t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
	}

	// give another goroutine a chance to run before we do a non-blocking read on a channel.
	runtime.Gosched()

	// make sure nothing comes through on the events channel
	select {
	case <-d.Events():
		t.Errorf("unexpected update to view with errored source")
	default:
	}
}

func TestBlankSourceWatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	b := Blank{}
	type basicConf struct {
		A int
		B int
		C string
	}
	c := basicConf{
		A: 3,
		B: 4809,
		C: "fob",
	}
	d, err := dials.Config(ctx, &c, &b)
	if err != nil {
		t.Fatalf("failed to construct View: %s", err)
	}
	initConf := d.View()
	if *initConf != c {
		t.Errorf("unexpected initial config: got %+v; expected %+v", *initConf, c)
	}

	triv := trivalCountingWatchingSource{}

	setErr := b.SetSource(ctx, &triv)
	if setErr != nil {
		t.Errorf("b.SetSource() failed with trivial nop impl: %s", setErr)
	}

	{
		// Await the new value, since it's sent asynchronously
		newConf := <-d.Events()
		if *newConf != c {
			t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, c)
		}
		if triv.callCount != 1 {
			t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
		}
	}

	// Poke the the watching source, and verify that we get something back
	triv.poke(ctx)
	{
		// Await the new value, since it's sent asynchronously
		newConf := <-d.Events()
		if *newConf != c {
			t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, c)
		}
		if triv.callCount != 1 {
			t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
		}
		if !triv.watchcalled {
			t.Errorf("watch not called")
		}
	}

}
func TestBlankSourceErrorWatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	b := Blank{}
	type basicConf struct {
		A int
		B int
		C string
	}
	c := basicConf{
		A: 3,
		B: 4809,
		C: "fob",
	}
	d, err := dials.Config(ctx, &c, &b)
	if err != nil {
		t.Fatalf("failed to construct View: %s", err)
	}
	initConf := d.View()
	if *initConf != c {
		t.Errorf("unexpected initial config: got %+v; expected %+v", *initConf, c)
	}

	expErr := errors.New("foobarbaz")
	triv := trivalErroringWatchingSource{
		err: expErr,
	}

	setErr := b.SetSource(ctx, &triv)
	if setErr == nil {
		t.Errorf("b.SetSource() did not fail as expected with erroring source")
	} else if wErr, ok := setErr.(*wrappedErr); !ok || wErr.Unwrap() != expErr {
		t.Errorf("b.SetSource() failed with with an unexpected error: %s", setErr)
	}

	{
		// Pull the new value (it should be immediately available)
		newConf := d.View()
		if *newConf != c {
			t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, c)
		}
		if triv.callCount != 1 {
			t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
		}
	}
}

type basicConfVerified struct {
	A int
	B int
	C string
}

func (b basicConfVerified) Verify() error {
	if b.A < -2 {
		return fmt.Errorf("value of A is too low: %d (< -2)", b.A)
	}
	return nil
}

// returns an empty value, but increments a counter for every call to Value()
// (not thread-safe)
type valuedCountingWatchingSource[V any] struct {
	callCount   uint32
	watchcalled bool
	args        dials.WatchArgs
	typ         *dials.Type
	val         V
}

var _ dials.Watcher = (*valuedCountingWatchingSource[struct{}])(nil)

func (t *valuedCountingWatchingSource[V]) Watch(ctx context.Context, typ *dials.Type, args dials.WatchArgs) error {
	t.watchcalled = true
	t.args = args
	t.typ = typ
	return nil
}

func (t *valuedCountingWatchingSource[V]) Value(_ context.Context, typ *dials.Type) (reflect.Value, error) {
	t.callCount++
	return reflect.ValueOf(t.val), nil
}

func (t *valuedCountingWatchingSource[V]) poke(ctx context.Context) {
	t.args.ReportNewValue(ctx, reflect.New(t.typ.Type()))
}

func TestBlankSourceWatcherVerifySkip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	b := Blank{}
	c := basicConfVerified{
		A: 3,
		B: 4809,
		C: "fob",
	}
	d, err := dials.Config(ctx, &c, &b)
	if err != nil {
		t.Fatalf("failed to construct View: %s", err)
	}
	initConf := d.View()
	if *initConf != c {
		t.Errorf("unexpected initial config: got %+v; expected %+v", *initConf, c)
	}

	type ptredBasicConf struct {
		A *int
		B *int
		C *string
	}
	n5val := -5
	triv := valuedCountingWatchingSource[ptredBasicConf]{
		val: ptredBasicConf{
			A: &n5val, // less than the error-threshold
		},
	}

	setErr := b.SetSourceSkipVerify(ctx, &triv)
	if setErr != nil {
		t.Errorf("b.SetSource() failed with trivial nop impl: %s", setErr)
	}

	{
		expConf := basicConfVerified{
			A: -5,
			B: 4809,
			C: "fob",
		}
		// Await the new value, since it's sent asynchronously
		newConf := <-d.Events()
		if *newConf != expConf {
			t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, expConf)
		}
		if triv.callCount != 1 {
			t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
		}
	}

	// Poke the the watching source, and verify that we get something back
	triv.poke(ctx)
	{
		// Await the new value, since it's sent asynchronously
		newConf := <-d.Events()
		if *newConf != c {
			t.Errorf("unexpected new config: got %+v; expected %+v", *newConf, c)
		}
		if triv.callCount != 1 {
			t.Errorf("unexpected call-count: %d; expected 1", triv.callCount)
		}
		if !triv.watchcalled {
			t.Errorf("watch not called")
		}
	}

}
