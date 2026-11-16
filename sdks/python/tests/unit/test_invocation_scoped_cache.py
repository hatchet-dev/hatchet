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


def test_callback_cache_index_tracks_removals() -> None:
    evicted: list[tuple[Key, str]] = []
    cache = DurableInvocationCallbackCache[str](
        on_evict=lambda key, value: evicted.append((key, value))
    )

    cache[("task-1", 1, 1, 1)] = "popped"
    cache[("task-1", 1, 1, 2)] = "deleted"
    cache[("task-1", 1, 1, 3)] = "kept"
    cache[("task-2", 1, 1, 1)] = "other task"

    assert cache.pop(("task-1", 1, 1, 1)) == "popped"
    del cache[("task-1", 1, 1, 2)]
    assert cache.pop(("task-1", 9, 9, 9), "missing") == "missing"

    cache[("task-1", 2, 1, 1)] = "newer"

    assert evicted == [(("task-1", 1, 1, 3), "kept")]
    assert set(cache) == {("task-1", 2, 1, 1), ("task-2", 1, 1, 1)}

    cache.clear()
    cache[("task-1", 1, 1, 1)] = "after clear"
    assert len(evicted) == 1
