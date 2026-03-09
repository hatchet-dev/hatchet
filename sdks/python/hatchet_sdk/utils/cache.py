import asyncio
from collections import OrderedDict
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
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


class TTLCache(dict[K, V]):
    def __init__(self, ttl: timedelta) -> None:
        self.ttl = ttl
        self.cache: dict[K, TTLCacheEntry[V]] = {}

        self.eviction_job = asyncio.create_task(self._start_eviction_job())

    def __setitem__(self, key: K, value: V) -> None:
        self.cache[key] = TTLCacheEntry(
            value=value, expires_at=datetime.now(tz=UTC) + self.ttl
        )

    def __getitem__(self, key: K) -> V:
        return self.cache[key].value

    async def _start_eviction_job(self) -> None:
        while True:
            now = datetime.now(tz=UTC)

            for key, entry in self.cache.items():
                if entry.expires_at <= now:
                    del self.cache[key]

            await asyncio.sleep(self.ttl.total_seconds())
