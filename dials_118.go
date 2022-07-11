//go:build !go1.19

package dials

import (
	"sync/atomic"
)

// Dials is the main access point for your configuration.
type Dials[T any] struct {
	value       atomic.Value
	updatesChan chan *T
	params      Params[T]
	cbch        chan<- userCallbackEvent
}

// View returns the configuration struct populated.
func (d *Dials[T]) View() *T {
	v, _ := d.value.Load().(*versionedConfig[T])
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return v.cfg
}

// View returns the configuration struct populated, and an opaque token.
func (d *Dials[T]) ViewVersion() (*T, CfgSerial[T]) {
	v, _ := d.value.Load().(*versionedConfig[T])
	// v cannot be nil because we initialize this value immediately after
	// creating the the Dials object
	return v.cfg, CfgSerial[T]{s: v.serial, cfg: v.cfg}
}
