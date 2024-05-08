package types

type RateLimitDuration string

const (
	Second RateLimitDuration = "second"
	Minute RateLimitDuration = "minute"
	Hour   RateLimitDuration = "hour"
	Day    RateLimitDuration = "day"
	Week   RateLimitDuration = "week"
	Month  RateLimitDuration = "month"
	Year   RateLimitDuration = "year"
)

type RateLimitOpts struct {
	Max      int
	Duration RateLimitDuration
}
