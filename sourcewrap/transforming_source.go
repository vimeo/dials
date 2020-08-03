package sourcewrap

import (
	"context"
	"io"
	"reflect"

	"github.com/vimeo/dials"
	"github.com/vimeo/dials/transform"
)

// NewTransformingDecoder constructs a dials.Decoder wrapping a slice of `transform.Mangler`s and another decoder.
func NewTransformingDecoder(dec dials.Decoder, manglers ...transform.Mangler) dials.Decoder {
	return &transformingDecoder{
		manglers: manglers,
		inner:    dec,
	}
}

type transformingDecoder struct {
	manglers []transform.Mangler
	inner    dials.Decoder
}

func (t *transformingDecoder) Decode(reader io.Reader, typ *dials.Type) (reflect.Value, error) {
	tfm := transform.NewTransformer(typ.Type(), t.manglers...)
	transformedVal, transformErr := tfm.TranslateType()
	if transformErr != nil {
		return reflect.Value{}, transformErr
	}
	innerTyp := dials.NewType(transformedVal)

	srcVal, srcErr := t.inner.Decode(reader, innerTyp)
	if srcErr != nil {
		return reflect.Value{}, &wrappedErr{prefix: "inner source failed: ", err: srcErr}
	}

	retVal, revErr := tfm.ReverseTranslate(srcVal)
	if revErr != nil {
		return reflect.Value{}, &wrappedErr{prefix: "inner reverse translate failed: ", err: revErr}
	}

	return retVal, nil
}

// NewTransformingSource constructs a dials.Source wrapping a slice of
// `transform.Mangler`s and another source. It picks the underlying
// implementation based on whether the wrapped source implements the Watcher
// interface to preserve that property.
func NewTransformingSource(src dials.Source, manglers ...transform.Mangler) dials.Source {
	nowatchTransformer := transformingSourceNoWatch{
		manglers: manglers,
		src:      src,
	}
	if watcher, ok := src.(dials.Watcher); ok {
		return &transformingSourceWithWatch{
			transformingSourceNoWatch: nowatchTransformer,
			src:                       watcher,
		}
	}
	return &nowatchTransformer

}

// transformingSource wraps a slice of `transformer.Mangler`s and another source
type transformingSourceNoWatch struct {
	manglers []transform.Mangler
	src      dials.Source
}

func (t *transformingSourceNoWatch) Value(typ *dials.Type) (reflect.Value, error) {
	tfm := transform.NewTransformer(typ.Type(), t.manglers...)
	transformedVal, transformErr := tfm.TranslateType()
	if transformErr != nil {
		return reflect.Value{}, &wrappedErr{prefix: "transform failed: ", err: transformErr}
	}
	innerTyp := dials.NewType(transformedVal)

	srcVal, srcErr := t.src.Value(innerTyp)
	if srcErr != nil {
		return reflect.Value{}, &wrappedErr{prefix: "inner source failed: ", err: srcErr}
	}
	unmangledVal, unmangleErr := tfm.ReverseTranslate(srcVal)
	if unmangleErr != nil {
		return reflect.Value{}, &wrappedErr{prefix: "unmangle failed: ", err: unmangleErr}
	}
	return unmangledVal, nil

}

type transformingSourceWithWatch struct {
	// embed the watch-less version
	transformingSourceNoWatch
	src dials.Watcher
}

func (t *transformingSourceWithWatch) Watch(ctx context.Context, typ *dials.Type, cb func(context.Context, reflect.Value)) error {
	tfm := transform.NewTransformer(typ.Type(), t.manglers...)
	transformedVal, transformErr := tfm.TranslateType()
	if transformErr != nil {
		return &wrappedErr{prefix: "transform failed: ", err: transformErr}
	}
	wrappedCB := func(ctx context.Context, val reflect.Value) {
		unmangledVal, unmangleErr := tfm.ReverseTranslate(val)
		if unmangleErr != nil {
			// TODO: update Watcher interface so cb returns an
			// error (which could also function to indicate to the
			// watching-source that the context was View's context
			// was cancelled or the local-context was cancelled)
			return
		}
		cb(ctx, unmangledVal)
	}
	innerTyp := dials.NewType(transformedVal)
	srcWatchErr := t.src.Watch(ctx, innerTyp, wrappedCB)
	if srcWatchErr != nil {
		return &wrappedErr{prefix: "failed to setup watcher: ", err: srcWatchErr}
	}
	return nil
}
