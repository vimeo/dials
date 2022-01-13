package dials

import (
	"context"
	"fmt"
)

type callbackMgr struct {
	p *Params

	ch <-chan userCallbackEvent
}

type userCallbackEvent interface {
	isUserCallbackEvent()
}

type newConfigEvent struct {
	oldConfig, newConfig interface{}
}

func (*newConfigEvent) isUserCallbackEvent() {}

var _ userCallbackEvent = (*newConfigEvent)(nil)

// watchErrorEvent sends the arguments to an OnWatchedError callback. The
// fields here must stay in sync with the arguments to WatchedErrorHandler.
type watchErrorEvent struct {
	err                  error
	oldConfig, newConfig interface{}
}

func (*watchErrorEvent) isUserCallbackEvent() {}

var _ userCallbackEvent = (*watchErrorEvent)(nil)

func (cbm *callbackMgr) runCBs(ctx context.Context) {
	for ev := range cbm.ch {
		switch e := ev.(type) {
		case *watchErrorEvent:
			if cbm.p.OnWatchedError != nil {
				cbm.p.OnWatchedError(ctx, e.err, e.oldConfig, e.newConfig)
			}
		case *newConfigEvent:
			if cbm.p.OnNewConfig != nil {
				cbm.p.OnNewConfig(ctx, e.oldConfig, e.newConfig)
			}
		default:
			panic(fmt.Errorf("unknown type %T for user callback event", ev))
		}
	}
}
