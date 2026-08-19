from __future__ import annotations

from hatchet_sdk.utils.cache import DurableInvocationCallbackCache

Key = tuple[str, int, int, int]


def test_callback_cache() -> None:
    evicted: list[tuple[Key, str]] = []
    cache = DurableInvocationCallbackCache[str](
        on_evict=lambda key, value: evicted.append((key, value))
    )

    cache[("task-1", 1, 1, 1)] = "old"
    cache[("task-1", 1, 1, 2)] = "old"
    cache[("task-1", 1, 1, 3)] = "new"

    assert len(cache) == 3
    assert len(evicted) == 0

    cache[("task-1", 2, 1, 1)] = "newer"

    assert len(cache) == 1
    assert cache[("task-1", 2, 1, 1)] == "newer"
    assert len(evicted) == 3
