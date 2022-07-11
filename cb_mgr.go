package dials

import (
	"context"
	"fmt"
)

type callbackMgr[T any] struct {
	p *Params[T]

	ch <-chan userCallbackEvent
}

type userCallbackEvent interface {
	isUserCallbackEvent()
}

type newConfigEvent[T any] struct {
	oldConfig, newConfig *T
}

func (*newConfigEvent[T]) isUserCallbackEvent() {}

var _ userCallbackEvent = (*newConfigEvent[struct{}])(nil)

// watchErrorEvent sends the arguments to an OnWatchedError callback. The
// fields here must stay in sync with the arguments to WatchedErrorHandler.
type watchErrorEvent[T any] struct {
	err                  error
	oldConfig, newConfig *T
}

func (*watchErrorEvent[T]) isUserCallbackEvent() {}

var _ userCallbackEvent = (*watchErrorEvent[struct{}])(nil)

func (cbm *callbackMgr[T]) runCBs(ctx context.Context) {
	for ev := range cbm.ch {
		switch e := ev.(type) {
		case *watchErrorEvent[T]:
			if cbm.p.OnWatchedError != nil {
				cbm.p.OnWatchedError(ctx, e.err, e.oldConfig, e.newConfig)
			}
		case *newConfigEvent[T]:
			if cbm.p.OnNewConfig != nil {
				cbm.p.OnNewConfig(ctx, e.oldConfig, e.newConfig)
			}
		default:
			panic(fmt.Errorf("unknown type %T for user callback event", ev))
		}
	}
}
