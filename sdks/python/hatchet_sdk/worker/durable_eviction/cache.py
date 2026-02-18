from __future__ import annotations

from datetime import datetime, timedelta
from typing import Protocol

from pydantic import BaseModel, ConfigDict

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.runnables.eviction import EvictionPolicy


class DurableRunRecord(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    key: ActionKey
    step_run_id: str
    token: CancellationToken
    eviction: EvictionPolicy | None
    registered_at: datetime

    # Waiting state
    waiting_since: datetime | None = None
    wait_kind: str | None = None
    wait_resource_id: str | None = None

    def is_waiting(self) -> bool:
        return self.waiting_since is not None


class DurableEvictionCache(Protocol):
    def register_run(
        self,
        key: ActionKey,
        step_run_id: str,
        token: CancellationToken,
        *,
        now: datetime,
        eviction: EvictionPolicy | None,
    ) -> None: ...

    def unregister_run(self, key: ActionKey) -> None: ...

    def mark_waiting(
        self,
        key: ActionKey,
        *,
        now: datetime,
        wait_kind: str,
        resource_id: str,
    ) -> None: ...

    def mark_active(self, key: ActionKey, *, now: datetime) -> None: ...

    def select_eviction_candidate(
        self,
        *,
        now: datetime,
        durable_slots: int,
        reserve_slots: int,
        min_wait_for_capacity_eviction: timedelta,
    ) -> ActionKey | None: ...

    def get(self, key: ActionKey) -> DurableRunRecord | None: ...


class InMemoryDurableEvictionCache(DurableEvictionCache):
    def __init__(self) -> None:
        self._runs: dict[ActionKey, DurableRunRecord] = {}

    def register_run(
        self,
        key: ActionKey,
        step_run_id: str,
        token: CancellationToken,
        *,
        now: datetime,
        eviction: EvictionPolicy | None,
    ) -> None:
        self._runs[key] = DurableRunRecord(
            key=key,
            step_run_id=step_run_id,
            token=token,
            eviction=eviction,
            registered_at=now,
        )

    def unregister_run(self, key: ActionKey) -> None:
        self._runs.pop(key, None)

    def get(self, key: ActionKey) -> DurableRunRecord | None:
        return self._runs.get(key)

    def mark_waiting(
        self,
        key: ActionKey,
        *,
        now: datetime,
        wait_kind: str,
        resource_id: str,
    ) -> None:
        rec = self._runs.get(key)
        if not rec:
            return

        # If already cancelled, don't bother tracking.
        if rec.token.is_cancelled:
            return

        rec.waiting_since = now
        rec.wait_kind = wait_kind
        rec.wait_resource_id = resource_id

    def mark_active(self, key: ActionKey, *, now: datetime) -> None:
        rec = self._runs.get(key)
        if not rec:
            return

        # Clear waiting state
        rec.waiting_since = None
        rec.wait_kind = None
        rec.wait_resource_id = None

    def _capacity_pressure(
        self, *, durable_slots: int, reserve_slots: int, waiting_count: int
    ) -> bool:
        if durable_slots <= 0:
            return False

        max_waiting = durable_slots - reserve_slots
        if max_waiting <= 0:
            return False

        return waiting_count >= max_waiting

    def select_eviction_candidate(
        self,
        *,
        now: datetime,
        durable_slots: int,
        reserve_slots: int,
        min_wait_for_capacity_eviction: timedelta,
    ) -> ActionKey | None:
        waiting: list[DurableRunRecord] = [
            r
            for r in self._runs.values()
            if r.is_waiting() and not r.token.is_cancelled and r.eviction is not None
        ]

        if not waiting:
            return None

        # Prefer TTL-eligible candidates first.
        ttl_eligible: list[DurableRunRecord] = []
        for r in waiting:
            ttl = r.eviction.ttl if r.eviction else None
            if ttl is None or r.waiting_since is None:
                continue
            if (now - r.waiting_since) >= ttl:
                ttl_eligible.append(r)

        if ttl_eligible:
            ttl_eligible.sort(
                key=lambda r: (
                    r.eviction.priority if r.eviction else 0,
                    r.waiting_since or now,
                )
            )
            chosen = ttl_eligible[0]
            logger.debug(
                "DurableEvictionCache: TTL eviction candidate selected "
                f"step_run_id={chosen.step_run_id} kind={chosen.wait_kind}"
            )
            return chosen.key

        # Capacity eviction: only if we're above waiting capacity and run allows it.
        capacity_pressure = self._capacity_pressure(
            durable_slots=durable_slots,
            reserve_slots=reserve_slots,
            waiting_count=len(waiting),
        )
        if not capacity_pressure:
            return None

        capacity_candidates: list[DurableRunRecord] = []
        for r in waiting:
            if not r.eviction or not r.eviction.allow_capacity_eviction:
                continue
            if r.waiting_since is None:
                continue
            if (now - r.waiting_since) < min_wait_for_capacity_eviction:
                continue
            capacity_candidates.append(r)

        if not capacity_candidates:
            return None

        capacity_candidates.sort(
            key=lambda r: (
                r.eviction.priority if r.eviction else 0,
                r.waiting_since or now,
            )
        )
        chosen = capacity_candidates[0]
        logger.debug(
            "DurableEvictionCache: capacity eviction candidate selected "
            f"step_run_id={chosen.step_run_id} kind={chosen.wait_kind}"
        )
        return chosen.key
