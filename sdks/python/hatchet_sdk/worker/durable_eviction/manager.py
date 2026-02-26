from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from datetime import datetime, timedelta, timezone

from pydantic import BaseModel, ConfigDict

from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.cache import (
    DurableEvictionCache,
    DurableRunRecord,
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
        cancel_local: Callable[[ActionKey], None],
        request_eviction_with_ack: Callable[
            [ActionKey, DurableRunRecord], Awaitable[None]
        ],
        config: DurableEvictionConfig = DEFAULT_DURABLE_EVICTION_CONFIG,
        cache: DurableEvictionCache | None = None,
    ) -> None:
        self._durable_slots = durable_slots
        self._cancel_local = cancel_local
        self._request_eviction_with_ack = request_eviction_with_ack
        self._config = config
        self._cache = cache or DurableEvictionCache()

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
        eviction_policy: EvictionPolicy | None,
    ) -> None:
        self._cache.register_run(
            key,
            step_run_id,
            now=self._now(),
            eviction_policy=eviction_policy,
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

        try:
            while True:
                await asyncio.sleep(interval)
                await self._tick_safe()
        except asyncio.CancelledError:
            return

    async def _tick_safe(self) -> None:
        try:
            await self._tick()
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

                if rec.eviction_policy is None:
                    continue

                logger.debug(
                    "DurableEvictionManager: evicting durable run "
                    f"task_run_external_id={rec.step_run_id} wait_kind={rec.wait_kind} "
                    f"resource_id={rec.wait_resource_id} ttl={rec.eviction_policy.ttl} "
                    f"capacity_allowed={rec.eviction_policy.allow_capacity_eviction}"
                )

                await self._request_eviction_with_ack(key, rec)

                self._cancel_local(key)
                self.unregister_run(key)

    async def evict_all_waiting(self) -> int:
        """Evict every currently-waiting durable run. Used during graceful shutdown."""
        self.stop()

        waiting = self._cache.get_all_waiting()
        evicted = 0

        for rec in waiting:
            if rec.eviction_policy is None:
                continue

            logger.debug(
                "DurableEvictionManager: shutdown-evicting durable run "
                f"task_run_external_id={rec.step_run_id} wait_kind={rec.wait_kind} "
                f"resource_id={rec.wait_resource_id}"
            )

            try:
                await self._request_eviction_with_ack(rec.key, rec)
            except Exception:
                logger.exception(
                    f"DurableEvictionManager: failed to send eviction for "
                    f"step_run_id={rec.step_run_id}"
                )
                continue

            self._cancel_local(rec.key)
            self.unregister_run(rec.key)
            evicted += 1

        return evicted

    def _now(self) -> datetime:
        return datetime.now(timezone.utc)
