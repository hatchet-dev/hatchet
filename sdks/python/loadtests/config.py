from __future__ import annotations

import os
from dataclasses import dataclass, field


def _env_int(key: str, default: int) -> int:
    return int(os.environ.get(key, default))


@dataclass(frozen=True)
class LoadTestConfig:
    n_durable_sleep: int = field(
        default_factory=lambda: _env_int("LOAD_N_DURABLE_SLEEP", 1000)
    )
    n_durable_event: int = field(
        default_factory=lambda: _env_int("LOAD_N_DURABLE_EVENT", 500)
    )
    n_durable_child: int = field(
        default_factory=lambda: _env_int("LOAD_N_DURABLE_CHILD", 200)
    )
    n_eviction: int = field(default_factory=lambda: _env_int("LOAD_N_EVICTION", 1000))
    n_mixed: int = field(default_factory=lambda: _env_int("LOAD_N_MIXED", 500))
    n_throughput: int = field(
        default_factory=lambda: _env_int("LOAD_N_THROUGHPUT", 1000)
    )

    poll_concurrency: int = field(
        default_factory=lambda: _env_int("LOAD_POLL_CONCURRENCY", 50)
    )
    result_concurrency: int = field(
        default_factory=lambda: _env_int("LOAD_RESULT_CONCURRENCY", 200)
    )

    max_growth_mb: int = field(
        default_factory=lambda: _env_int("LOAD_MAX_GROWTH_MB", 200)
    )
