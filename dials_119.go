//go:build go1.19

package dials

import (
	"sync/atomic"
)

// Dials is the main access point for your configuration.
type Dials[T any] struct {
	value       atomic.Pointer[versionedConfig[T]]
	updatesChan chan *T
	params      Params[T]
	cbch        chan<- userCallbackEvent
	monCtl      chan<- verifyEnable[T]
}

// View returns the configuration struct populated.
func (d *Dials[T]) View() *T {
	versioned := d.value.Load()
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return versioned.cfg
}

// View returns the configuration struct populated, and an opaque token.
func (d *Dials[T]) ViewVersion() (*T, CfgSerial[T]) {
	versioned := d.value.Load()
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return versioned.cfg, CfgSerial[T]{s: versioned.serial, cfg: versioned.cfg}

}
