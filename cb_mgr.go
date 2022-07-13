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
	serial               uint64
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

type userCallbackHandle[T any] struct {
	cb        NewConfigHandler[T]
	minSerial uint64
}

type userCallbackRegistration[T any] struct {
	handle *userCallbackHandle[T]
	serial *CfgSerial[T]
}

func (*userCallbackRegistration[T]) isUserCallbackEvent() {}

var _ userCallbackEvent = (*userCallbackRegistration[struct{}])(nil)

type userCallbackUnregister[T any] struct {
	// handle describes the relevant callback, and is the key in the newCfgCBs set tracked by
	// runCBs.
	handle *userCallbackHandle[T]
	// done is closed by runCBs immediately after it's removed the handle from its set of user
	// callbacks to run.
	done chan<- struct{}
}

func (*userCallbackUnregister[T]) isUserCallbackEvent() {}

var _ userCallbackEvent = (*userCallbackUnregister[struct{}])(nil)

func (cbm *callbackMgr[T]) runCBs(ctx context.Context) {
	newCfgCBs := make(map[*userCallbackHandle[T]]struct{}, 0)
	lastSerial := uint64(0)
	lastVersion := (*T)(nil)
	for ev := range cbm.ch {
		switch e := ev.(type) {
		case *watchErrorEvent[T]:
			if cbm.p.OnWatchedError != nil {
				cbm.p.OnWatchedError(ctx, e.err, e.oldConfig, e.newConfig)
			}
		case *newConfigEvent[T]:
			lastSerial = e.serial
			lastVersion = e.newConfig
			if cbm.p.OnNewConfig != nil {
				cbm.p.OnNewConfig(ctx, e.oldConfig, e.newConfig)
			}
			for cbh := range newCfgCBs {
				if cbh.minSerial >= e.serial {
					// Skip the callback if it was registered with a serial for
					// a version that we haven't caught up to yet.
					continue
				}
				cbh.cb(ctx, e.oldConfig, e.newConfig)
			}
		case *userCallbackRegistration[T]:
			// Serial values are assigned sequentially, so make sure we don't deliver an
			// older config if we've fallen behind.
			if e.serial.cfg != nil && e.serial.s < lastSerial {
				e.handle.cb(ctx, e.serial.cfg, lastVersion)
			}
			// add this callback to the set of callbacks
			newCfgCBs[e.handle] = struct{}{}
		case *userCallbackUnregister[T]:
			delete(newCfgCBs, e.handle)
			close(e.done)
		default:
			panic(fmt.Errorf("unknown type %T for user callback event", ev))
		}
	}
}
