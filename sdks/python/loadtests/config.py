from dataclasses import dataclass


@dataclass
class LoadTestConfig:
    concurrency: int
    duration_seconds: int
    runs_per_second: int
    wait_seconds: int
    avg_threshold_ms: int
