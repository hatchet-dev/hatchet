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


DurableInvocationCallbackKey = tuple[str, int, int, int]


class DurableInvocationCallbackCache(dict[DurableInvocationCallbackKey, V]):
    def __init__(
        self, on_evict: Callable[[DurableInvocationCallbackKey, V], None] | None = None
    ) -> None:
        super().__init__()
        self._on_evict = on_evict
        self._keys_by_task_and_invocation: dict[
            str, dict[int, set[DurableInvocationCallbackKey]]
        ] = {}

    def __setitem__(self, key: DurableInvocationCallbackKey, value: V) -> None:
        task_external_id, invocation_count = key[0], key[1]
        invocations = self._keys_by_task_and_invocation.setdefault(task_external_id, {})

        superseded_invocations = [inv for inv in invocations if inv < invocation_count]
        for stale_invocation in superseded_invocations:
            for stale_key in invocations.pop(stale_invocation):
                stale_value = super().pop(stale_key)
                if self._on_evict is not None:
                    self._on_evict(stale_key, stale_value)

        invocations.setdefault(invocation_count, set()).add(key)
        super().__setitem__(key, value)

    def __delitem__(self, key: DurableInvocationCallbackKey) -> None:
        super().__delitem__(key)
        self._forget_key(key)

    def pop(self, key: DurableInvocationCallbackKey, *default: V) -> V:  # type: ignore[override]
        if key not in self:
            if default:
                return default[0]
            raise KeyError(key)
        value = super().pop(key)
        self._forget_key(key)
        return value

    def clear(self) -> None:
        super().clear()
        self._keys_by_task_and_invocation.clear()

    def _forget_key(self, key: DurableInvocationCallbackKey) -> None:
        task_external_id, invocation_count = key[0], key[1]
        invocations = self._keys_by_task_and_invocation.get(task_external_id)
        if invocations is None:
            return
        keys = invocations.get(invocation_count)
        if keys is None:
            return
        keys.discard(key)
        if not keys:
            del invocations[invocation_count]
        if not invocations:
            del self._keys_by_task_and_invocation[task_external_id]
