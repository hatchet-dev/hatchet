package types

type RateLimitDuration string

const (
	Second RateLimitDuration = "second"
	Minute RateLimitDuration = "minute"
	Hour   RateLimitDuration = "hour"
)

type RateLimitOpts struct {
	Max      int
	Duration RateLimitDuration
}
