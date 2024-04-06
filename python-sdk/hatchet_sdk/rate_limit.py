from dataclasses import dataclass


@dataclass
class RateLimit:
    key: str
    units: int


class RateLimitDuration:
    SECOND = "SECOND"
    MINUTE = "MINUTE"
    HOUR = "HOUR"
