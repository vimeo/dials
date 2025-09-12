package flaghelper

import (
	"time"
)

// TimeWrapper wraps a time.Time
//
// This is needed for the flag package in order to correctly
// print or omit the default value in PrintDefaults.
type TimeWrapper struct {
	t time.Time
}

// NewTimeWrapper creates a new TimeWrapper for a time.Time
func NewTimeWrapper(t time.Time) *TimeWrapper {
	return &TimeWrapper{
		t: t,
	}
}

// Set implements flag.Value
func (tw *TimeWrapper) Set(s string) error {
	return tw.t.UnmarshalText([]byte(s))
}

// Get implements flag.Value
func (tw *TimeWrapper) Get() any {
	return tw.t
}

// String implements flag.Value
func (tw *TimeWrapper) String() string {
	// This uses the same format as MarshalText but without the date range validation
	return tw.t.Format(time.RFC3339Nano)
}
