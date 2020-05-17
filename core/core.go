package core

import (
	"time"
)

// UnmarshalError should be returned for unmarshalling errors
type UnmarshalError error
// MarshalError should be returned for marshalling errors
type MarshalError error

// Value interface represents ways to serialize and deserialize data types
type Value interface {
	// UnmarshalValue should be implemented by self assigning and returning an error if not compatible
	UnmarshalValue(bytes []byte) error
	MarshalValue() ([]byte, error)
}

// Config represents an interface to implement that can return config
type Config interface {
	GetValue(path string, fallback Value) Value
	GetValueOrError(path string) (Value, error)
	MustGetValue(path string) Value

	Get(path string, fallback string) string
	GetOrError(path string) (string, error)
	MustGet(path string) string

	GetInt(path string, fallback int) int
	GetIntOrError(path string) (int, error)
	MustGetInt(path string) int

	GetUint(path string, fallback uint) uint
	GetUintOrError(path string) (uint, error)
	MustGetUint(path string) uint

	GetBool(path string, fallback bool) bool
	GetBoolOrError(path string) (bool, error)
	MustGetBool(path string) bool

	GetFloat(path string, fallback float64) float64
	GetFloatOrError(path string) (float64, error)
	MustGetFloat(path string) float64

	GetDuration(path string, fallback time.Duration) time.Duration
	GetDurationOrError(path string) (time.Duration, error)
	MustGetDuration(path string) time.Duration

	GetTime(path string, fallback time.Time) time.Time
	GetTimeOrError(path string) (time.Time, error)
	MustGetTime(path string) time.Time
}

// Matcher implements a simple matcher function type.
// the value is the input from the value field and should return (true, true) for a match, or (false, true) for a
// non-match.
//
// For optional matching, if it does not match, it should return (false, false)
// the results of all selectors are computed using AND.
type Matcher func(value []byte) (matched bool, required bool)


// FeatureFlag provides an interface to implement a feature flagging provider.
type FeatureFlag interface {
	// IsEnabled returns the boolean AND result of all selectors for the given path.
	// If all matchers are optional OR the path does not exist, the fallback value will be returned.
	IsEnabled(path string, fallback bool, matchers... string) bool
}

// KillSwitch implements an interface for kill switching providers.
type KillSwitch interface {
	// IsKilled returns the boolean AND result of all selectors for the given path.
	// If all matchers are optional OR the path does not exist, the fallback value will be returned.
	IsKilled(path string, fallback bool, matchers... string) bool
}
