from __future__ import annotations

from datetime import datetime, timedelta, timezone

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.cache import InMemoryDurableEvictionCache


def dt(seconds: int) -> datetime:
    return datetime(2026, 1, 1, 0, 0, 0, tzinfo=timezone.utc) + timedelta(
        seconds=seconds
    )


def test_ttl_eviction_prefers_oldest_waiting_and_priority() -> None:
    cache = InMemoryDurableEvictionCache()

    key1 = "run-1/0"
    key2 = "run-2/0"
    tok1 = CancellationToken()
    tok2 = CancellationToken()

    eviction_low_prio = EvictionPolicy(ttl=timedelta(seconds=10), priority=0)
    eviction_high_prio = EvictionPolicy(ttl=timedelta(seconds=10), priority=10)

    cache.register_run(key1, "run-1", tok1, now=dt(0), eviction=eviction_high_prio)
    cache.register_run(key2, "run-2", tok2, now=dt(0), eviction=eviction_low_prio)

    cache.mark_waiting(
        key1, now=dt(0), wait_kind="workflow_run_result", resource_id="wf1"
    )
    cache.mark_waiting(
        key2, now=dt(5), wait_kind="workflow_run_result", resource_id="wf2"
    )

    # Both are past TTL at now=20, but low priority should be evicted first.
    chosen = cache.select_eviction_candidate(
        now=dt(20),
        durable_slots=100,
        reserve_slots=0,
        min_wait_for_capacity_eviction=timedelta(seconds=0),
    )
    assert chosen == key2


def test_none_eviction_params_never_selected() -> None:
    cache = InMemoryDurableEvictionCache()

    key_no = "run-no/0"
    key_yes = "run-yes/0"

    cache.register_run(key_no, "run-no", CancellationToken(), now=dt(0), eviction=None)
    cache.register_run(
        key_yes,
        "run-yes",
        CancellationToken(),
        now=dt(0),
        eviction=EvictionPolicy(ttl=timedelta(seconds=1)),
    )

    cache.mark_waiting(key_no, now=dt(0), wait_kind="durable_event", resource_id="x")
    cache.mark_waiting(key_yes, now=dt(0), wait_kind="durable_event", resource_id="y")

    chosen = cache.select_eviction_candidate(
        now=dt(10),
        durable_slots=100,
        reserve_slots=0,
        min_wait_for_capacity_eviction=timedelta(seconds=0),
    )
    assert chosen == key_yes


def test_capacity_eviction_respects_allow_capacity_and_min_wait() -> None:
    cache = InMemoryDurableEvictionCache()

    key_blocked = "run-blocked/0"
    key_ok = "run-ok/0"

    cache.register_run(
        key_blocked,
        "run-blocked",
        CancellationToken(),
        now=dt(0),
        eviction=EvictionPolicy(
            ttl=timedelta(hours=1), allow_capacity_eviction=False, priority=0
        ),
    )
    cache.register_run(
        key_ok,
        "run-ok",
        CancellationToken(),
        now=dt(0),
        eviction=EvictionPolicy(
            ttl=timedelta(hours=1), allow_capacity_eviction=True, priority=0
        ),
    )

    cache.mark_waiting(
        key_blocked, now=dt(0), wait_kind="durable_event", resource_id="x"
    )
    cache.mark_waiting(key_ok, now=dt(0), wait_kind="durable_event", resource_id="y")

    # Capacity pressure because waiting_count==durable_slots==2, but enforce min-wait.
    chosen_too_soon = cache.select_eviction_candidate(
        now=dt(5),
        durable_slots=2,
        reserve_slots=0,
        min_wait_for_capacity_eviction=timedelta(seconds=10),
    )
    assert chosen_too_soon is None

    # Now past min wait: only key_ok is eligible for capacity eviction.
    chosen = cache.select_eviction_candidate(
        now=dt(15),
        durable_slots=2,
        reserve_slots=0,
        min_wait_for_capacity_eviction=timedelta(seconds=10),
    )
    assert chosen == key_ok
