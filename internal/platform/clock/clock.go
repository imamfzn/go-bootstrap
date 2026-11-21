// Package clock provides the current time behind an interface, so code that
// depends on time can be tested without sleeping.
package clock

import "time"

// Clock reports the current time.
type Clock interface {
	Now() time.Time
}

type system struct{}

func (system) Now() time.Time { return time.Now().UTC() }

// System returns a Clock backed by the wall clock, in UTC.
func System() Clock { return system{} }

// Fixed is a Clock that always reports the same instant. Use it in tests.
type Fixed time.Time

// Now returns the fixed instant.
func (f Fixed) Now() time.Time { return time.Time(f) }
