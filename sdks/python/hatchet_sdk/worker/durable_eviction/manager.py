from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from datetime import datetime, timedelta, timezone

from pydantic import BaseModel, ConfigDict

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.exceptions import CancellationReason
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.cache import (
    DurableEvictionCache,
    DurableRunRecord,
    InMemoryDurableEvictionCache,
)


class DurableEvictionConfig(BaseModel):
    model_config = ConfigDict(frozen=True)

    check_interval: timedelta = timedelta(seconds=1)
    """How often we try selecting an eviction candidate."""

    reserve_slots: int = 0
    """How many slots to reserve from capacity-based eviction decisions."""

    min_wait_for_capacity_eviction: timedelta = timedelta(seconds=10)
    """Avoid immediately evicting runs that have just entered a wait."""

DEFAULT_DURABLE_EVICTION_CONFIG = DurableEvictionConfig()


class DurableEvictionManager:
    def __init__(
        self,
        *,
        durable_slots: int,
        cancel_remote: Callable[[str], Awaitable[None]],
        on_eviction_selected: Callable[[ActionKey, DurableRunRecord], Awaitable[None]]
        | None = None,
        on_eviction_cancelled: Callable[[ActionKey, DurableRunRecord], Awaitable[None]]
        | None = None,
        config: DurableEvictionConfig = DEFAULT_DURABLE_EVICTION_CONFIG,
        cache: DurableEvictionCache | None = None,
    ) -> None:
        self._durable_slots = durable_slots
        self._cancel_remote = cancel_remote
        self._on_eviction_selected = on_eviction_selected
        self._on_eviction_cancelled = on_eviction_cancelled
        self._config = config
        self._cache: DurableEvictionCache = cache or InMemoryDurableEvictionCache()

        self._task: asyncio.Task[None] | None = None
        self._lock = asyncio.Lock()

    @property
    def cache(self) -> DurableEvictionCache:
        return self._cache

    def start(self) -> None:
        # Lazy start (requires an event loop)
        if self._task and not self._task.done():
            return
        self._task = asyncio.create_task(self._run_loop())

    def stop(self) -> None:
        if self._task and not self._task.done():
            self._task.cancel()

    def register_run(
        self,
        key: ActionKey,
        *,
        step_run_id: str,
        token: CancellationToken,
        eviction: EvictionPolicy | None,
    ) -> None:
        self._cache.register_run(
            key,
            step_run_id,
            token,
            now=self._now(),
            eviction=eviction,
        )

    def unregister_run(self, key: ActionKey) -> None:
        self._cache.unregister_run(key)

    def mark_waiting(
        self,
        key: ActionKey,
        *,
        wait_kind: str,
        resource_id: str,
    ) -> None:
        self._cache.mark_waiting(
            key,
            now=self._now(),
            wait_kind=wait_kind,
            resource_id=resource_id,
        )

    def mark_active(self, key: ActionKey) -> None:
        self._cache.mark_active(key, now=self._now())

    async def _run_loop(self) -> None:
        interval = self._config.check_interval.total_seconds()

        while True:
            try:
                await asyncio.sleep(interval)
                await self._tick()
            except asyncio.CancelledError:
                return
            except Exception:
                logger.exception("DurableEvictionManager: error in eviction loop")

    async def _tick(self) -> None:
        # Only one eviction *cycle* at a time.
        #
        # Within a tick we drain all currently-eligible candidates
        async with self._lock:
            evicted_this_tick: set[ActionKey] = set()

            while True:
                key = self._cache.select_eviction_candidate(
                    now=self._now(),
                    durable_slots=self._durable_slots,
                    reserve_slots=self._config.reserve_slots,
                    min_wait_for_capacity_eviction=self._config.min_wait_for_capacity_eviction,
                )
                if key is None:
                    return

                # Safety: avoid infinite loops if cache repeatedly returns same key.
                if key in evicted_this_tick:
                    return
                evicted_this_tick.add(key)

                rec = self._cache.get(key)
                if rec is None:
                    continue

                # Not evictable (shouldn't be selected, but be defensive).
                if rec.eviction is None:
                    continue

                logger.warning(
                    "DurableEvictionManager: evicting durable run "
                    f"step_run_id={rec.step_run_id} wait_kind={rec.wait_kind} "
                    f"resource_id={rec.wait_resource_id} ttl={rec.eviction.ttl} "
                    f"capacity_allowed={rec.eviction.allow_capacity_eviction}"
                )

                # Observability hook: emitted when selected from eviction cache.
                # Best-effort; eviction should proceed even if this fails.
                if self._on_eviction_selected is not None:
                    try:
                        await self._on_eviction_selected(key, rec)
                    except Exception:
                        logger.exception(
                            "DurableEvictionManager: error emitting eviction event"
                        )

                # TODO-DURABLE: the eviction event is not optional, and we need to ack it before unwinding locally.

                # Unwind locally ASAP (causes waits to raise). Do this *before* the remote
                # cancel so the CancellationToken reason remains `evicted` even if the
                # engine cancel action arrives quickly.
                rec.token.cancel(CancellationReason.EVICTED)

                # Observability hook: emitted when we actually cancel locally.
                # Best-effort; eviction should proceed even if this fails.
                if self._on_eviction_cancelled is not None:
                    try:
                        await self._on_eviction_cancelled(key, rec)
                    except Exception:
                        logger.exception(
                            "DurableEvictionManager: error emitting eviction cancellation event"
                        )

    def _now(self) -> datetime:
        return datetime.now(timezone.utc)

