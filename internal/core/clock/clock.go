package clock

import "time"

// Clock is a small abstraction over [time.Now] so tests can inject a fixed
// or advanceable time source.
type Clock interface {
	Now() time.Time
}

// Real reports the current wall-clock time.
type Real struct{}

// Now returns [time.Now].
func (Real) Now() time.Time { return time.Now() }

// Fake is a Clock whose value can be set and advanced by tests.
//
// Unlike Real, Fake uses a pointer receiver so that Advance mutates the
// stored time in place. Construct via &Fake{T: ...}.
type Fake struct {
	T time.Time
}

// Now returns the fake's current time.
func (f *Fake) Now() time.Time { return f.T }

// Advance moves the fake's time forward by d.
func (f *Fake) Advance(d time.Duration) { f.T = f.T.Add(d) }
