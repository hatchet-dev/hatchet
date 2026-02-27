from __future__ import annotations

from datetime import datetime, timedelta
from enum import Enum

try:
    from typing import assert_never
except ImportError:
    from typing_extensions import assert_never

from pydantic import BaseModel

from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.runnables.eviction import EvictionPolicy


class EvictionCause(str, Enum):
    TTL_EXCEEDED = "ttl_exceeded"
    CAPACITY_PRESSURE = "capacity_pressure"
    WORKER_SHUTDOWN = "worker_shutdown"


class DurableRunRecord(BaseModel):
    key: ActionKey
    step_run_id: str
    eviction_policy: EvictionPolicy | None
    registered_at: datetime

    # Waiting state
    waiting_since: datetime | None = None
    wait_kind: str | None = None
    wait_resource_id: str | None = None

    # Set by the eviction manager before requesting eviction
    eviction_reason: str | None = None

    @property
    def is_waiting(self) -> bool:
        return self.waiting_since is not None


class DurableEvictionCache:
    def __init__(self) -> None:
        self._runs: dict[ActionKey, DurableRunRecord] = {}

    def register_run(
        self,
        key: ActionKey,
        step_run_id: str,
        now: datetime,
        eviction_policy: EvictionPolicy | None,
    ) -> None:
        self._runs[key] = DurableRunRecord(
            key=key,
            step_run_id=step_run_id,
            eviction_policy=eviction_policy,
            registered_at=now,
        )

    def unregister_run(self, key: ActionKey) -> None:
        self._runs.pop(key, None)

    def get(self, key: ActionKey) -> DurableRunRecord | None:
        return self._runs.get(key)

    def get_all_waiting(self) -> list[DurableRunRecord]:
        return [r for r in self._runs.values() if r.is_waiting]

    def mark_waiting(
        self,
        key: ActionKey,
        now: datetime,
        wait_kind: str,
        resource_id: str,
    ) -> None:
        rec = self._runs.get(key)
        if not rec:
            return

        rec.waiting_since = now
        rec.wait_kind = wait_kind
        rec.wait_resource_id = resource_id

    def mark_active(self, key: ActionKey, now: datetime) -> None:
        rec = self._runs.get(key)
        if not rec:
            return

        # Clear waiting state
        rec.waiting_since = None
        rec.wait_kind = None
        rec.wait_resource_id = None

    def _capacity_pressure(
        self, durable_slots: int, reserve_slots: int, waiting_count: int
    ) -> bool:
        if durable_slots <= 0:
            return False

        max_waiting = durable_slots - reserve_slots
        if max_waiting <= 0:
            return False

        return waiting_count >= max_waiting

    def select_eviction_candidate(
        self,
        now: datetime,
        durable_slots: int,
        reserve_slots: int,
        min_wait_for_capacity_eviction: timedelta,
    ) -> ActionKey | None:
        waiting: list[DurableRunRecord] = [
            r
            for r in self._runs.values()
            if r.is_waiting and r.eviction_policy is not None
        ]

        if not waiting:
            return None

        # Prefer TTL-eligible candidates first.
        ttl_eligible: list[DurableRunRecord] = [
            r
            for r in waiting
            if r.eviction_policy is not None
            and r.eviction_policy.ttl is not None
            and r.waiting_since is not None
            and (now - r.waiting_since) >= r.eviction_policy.ttl
        ]

        if ttl_eligible:
            ttl_eligible.sort(
                key=lambda r: (
                    r.eviction_policy.priority if r.eviction_policy else 0,
                    r.waiting_since or now,
                )
            )
            chosen = ttl_eligible[0]
            ttl = chosen.eviction_policy.ttl if chosen.eviction_policy else None
            chosen.eviction_reason = _build_eviction_reason(
                EvictionCause.TTL_EXCEEDED, chosen, ttl=ttl
            )
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

        capacity_candidates: list[DurableRunRecord] = [
            r
            for r in waiting
            if r.eviction_policy
            and r.eviction_policy.allow_capacity_eviction
            and r.waiting_since is not None
            and (now - r.waiting_since) >= min_wait_for_capacity_eviction
        ]

        if not capacity_candidates:
            return None

        capacity_candidates.sort(
            key=lambda r: (
                r.eviction_policy.priority if r.eviction_policy else 0,
                r.waiting_since or now,
            )
        )
        chosen = capacity_candidates[0]
        chosen.eviction_reason = _build_eviction_reason(
            EvictionCause.CAPACITY_PRESSURE, chosen
        )
        logger.debug(
            "DurableEvictionCache: capacity eviction candidate selected "
            f"step_run_id={chosen.step_run_id} kind={chosen.wait_kind}"
        )
        return chosen.key


def _build_eviction_reason(
    cause: EvictionCause,
    rec: DurableRunRecord,
    ttl: timedelta | None = None,
) -> str:
    wait_desc = rec.wait_kind or "unknown"
    if rec.wait_resource_id:
        wait_desc = f"{wait_desc}({rec.wait_resource_id})"

    match cause:
        case EvictionCause.TTL_EXCEEDED:
            ttl_str = f" ({ttl})" if ttl else ""
            return f"Wait TTL{ttl_str} exceeded while waiting on {wait_desc}"
        case EvictionCause.CAPACITY_PRESSURE:
            return f"Worker at capacity while waiting on {wait_desc}"
        case EvictionCause.WORKER_SHUTDOWN:
            return f"Worker shutdown while waiting on {wait_desc}"
        case _ as unreachable:
            assert_never(unreachable)
