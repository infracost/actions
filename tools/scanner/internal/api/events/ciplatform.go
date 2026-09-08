package events

// CIPlatform returns the CI platform detected from the environment, the same
// value reported in the events payload.
func CIPlatform() string { return getCIPlatform() }
