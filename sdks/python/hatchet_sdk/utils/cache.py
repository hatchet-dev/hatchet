import asyncio
from collections import OrderedDict
from collections.abc import Callable, Iterator
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Generic, TypeVar

K = TypeVar("K")
V = TypeVar("V")


class BoundedDict(OrderedDict[K, V]):
    def __init__(self, maxsize: int):
        super().__init__()
        self.maxsize = maxsize

    def __setitem__(self, key: K, value: V) -> None:
        if key in self:
            self.move_to_end(key)

        super().__setitem__(key, value)

        if len(self) > self.maxsize:
            self.popitem(last=False)


@dataclass
class TTLCacheEntry(Generic[V]):
    value: V
    expires_at: datetime


class TTLCache(Generic[K, V]):
    def __init__(self, ttl: timedelta) -> None:
        self.ttl = ttl
        self.cache: dict[K, TTLCacheEntry[V]] = {}

        self.eviction_job = asyncio.create_task(self._start_eviction_job())

    def __setitem__(self, key: K, value: V) -> None:
        self.cache[key] = TTLCacheEntry(
            value=value, expires_at=datetime.now(tz=timezone.utc) + self.ttl
        )

    def __getitem__(self, key: K) -> V:
        return self.cache[key].value

    def __contains__(self, key: object) -> bool:
        return key in self.cache

    def __delitem__(self, key: K) -> None:
        del self.cache[key]

    def __iter__(self) -> Iterator[K]:
        return iter(self.cache)

    def pop(self, key: K) -> V:
        return self.cache.pop(key).value

    def clear(self) -> None:
        self.cache.clear()

    def stop_eviction_job(self) -> None:
        self.eviction_job.cancel()

    async def _start_eviction_job(self) -> None:
        while True:
            await asyncio.sleep(self.ttl.total_seconds())

            now = datetime.now(tz=timezone.utc)
            expired = [k for k, entry in self.cache.items() if entry.expires_at <= now]

            for key in expired:
                del self.cache[key]


class DurableInvocationCallbackCache(dict[tuple[str, int, int, int], V]):
    def __init__(
        self, on_evict: Callable[[tuple[str, int, int, int], V], None] | None = None
    ) -> None:
        super().__init__()
        self._on_evict = on_evict

    def __setitem__(self, key: tuple[str, int, int, int], value: V) -> None:
        task_external_id, invocation_count = key[0], key[1]

        superseded = [
            k for k in self if k[0] == task_external_id and k[1] < invocation_count
        ]

        for stale_key in superseded:
            stale_value = super().pop(stale_key)

            if self._on_evict is not None:
                self._on_evict(stale_key, stale_value)

        super().__setitem__(key, value)
