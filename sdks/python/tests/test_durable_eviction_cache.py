from __future__ import annotations

from datetime import datetime, timedelta, timezone

from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.cache import DurableEvictionCache


def dt(seconds: int) -> datetime:
    return datetime(2026, 1, 1, 0, 0, 0, tzinfo=timezone.utc) + timedelta(
        seconds=seconds
    )


def test_ttl_eviction_prefers_oldest_waiting_and_priority() -> None:
    cache = DurableEvictionCache()

    key1 = "run-1/0"
    key2 = "run-2/0"

    eviction_low_prio = EvictionPolicy(ttl=timedelta(seconds=10), priority=0)
    eviction_high_prio = EvictionPolicy(ttl=timedelta(seconds=10), priority=10)

    cache.register_run(key1, "run-1", invocation_count=1, now=dt(0), eviction_policy=eviction_high_prio)
    cache.register_run(key2, "run-2", invocation_count=1, now=dt(0), eviction_policy=eviction_low_prio)

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
    cache = DurableEvictionCache()

    key_no = "run-no/0"
    key_yes = "run-yes/0"

    cache.register_run(key_no, "run-no", invocation_count=1, now=dt(0), eviction_policy=None)
    cache.register_run(
        key_yes,
        "run-yes",
        invocation_count=1,
        now=dt(0),
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=1)),
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
    cache = DurableEvictionCache()

    key_blocked = "run-blocked/0"
    key_ok = "run-ok/0"

    cache.register_run(
        key_blocked,
        "run-blocked",
        invocation_count=1,
        now=dt(0),
        eviction_policy=EvictionPolicy(
            ttl=timedelta(hours=1), allow_capacity_eviction=False, priority=0
        ),
    )
    cache.register_run(
        key_ok,
        "run-ok",
        invocation_count=1,
        now=dt(0),
        eviction_policy=EvictionPolicy(
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


def test_concurrent_waits_keep_waiting_until_all_resolved() -> None:
    """Simulates asyncio.gather over 3 child aio_result() calls on the same run.

    When one child completes (mark_active), the run must remain in waiting
    state until *all* concurrent waits have resolved.
    """
    cache = DurableEvictionCache()
    key = "run-bulk/0"
    policy = EvictionPolicy(ttl=timedelta(seconds=5), priority=0)

    cache.register_run(key, "run-bulk", invocation_count=1, now=dt(0), eviction_policy=policy)

    cache.mark_waiting(key, now=dt(1), wait_kind="spawn_child", resource_id="child0")
    cache.mark_waiting(key, now=dt(1), wait_kind="spawn_child", resource_id="child1")
    cache.mark_waiting(key, now=dt(1), wait_kind="spawn_child", resource_id="child2")

    rec = cache.get(key)
    assert rec is not None
    assert rec.is_waiting
    assert rec._wait_count == 3

    # child0 completes -- run should still be waiting
    cache.mark_active(key, now=dt(2))
    assert rec.is_waiting
    assert rec._wait_count == 2
    assert rec.waiting_since == dt(1)

    # TTL still fires while 2 children are pending
    chosen = cache.select_eviction_candidate(
        now=dt(10),
        durable_slots=100,
        reserve_slots=0,
        min_wait_for_capacity_eviction=timedelta(seconds=0),
    )
    assert chosen == key

    # child1 completes
    cache.mark_active(key, now=dt(11))
    assert rec.is_waiting
    assert rec._wait_count == 1

    # child2 completes -- now the run is truly active
    cache.mark_active(key, now=dt(12))
    assert not rec.is_waiting
    assert rec._wait_count == 0
    assert rec.waiting_since is None


def test_find_key_by_step_run_id_returns_matching_key() -> None:
    cache = DurableEvictionCache()
    cache.register_run("run-a/0", "ext-a", invocation_count=1, now=dt(0), eviction_policy=None)
    cache.register_run("run-b/0", "ext-b", invocation_count=1, now=dt(0), eviction_policy=None)

    assert cache.find_key_by_step_run_id("ext-a") == "run-a/0"
    assert cache.find_key_by_step_run_id("ext-b") == "run-b/0"


def test_find_key_by_step_run_id_returns_none_for_unknown() -> None:
    cache = DurableEvictionCache()
    cache.register_run("run-a/0", "ext-a", invocation_count=1, now=dt(0), eviction_policy=None)

    assert cache.find_key_by_step_run_id("no-such-id") is None


def test_find_key_by_step_run_id_returns_none_after_unregister() -> None:
    cache = DurableEvictionCache()
    cache.register_run("run-a/0", "ext-a", invocation_count=1, now=dt(0), eviction_policy=None)

    assert cache.find_key_by_step_run_id("ext-a") == "run-a/0"
    cache.unregister_run("run-a/0")
    assert cache.find_key_by_step_run_id("ext-a") is None


def test_mark_active_floors_at_zero() -> None:
    """Extra mark_active calls (defensive) should not go negative."""
    cache = DurableEvictionCache()
    key = "run-extra/0"
    policy = EvictionPolicy(ttl=timedelta(seconds=5), priority=0)

    cache.register_run(key, "run-extra", invocation_count=1, now=dt(0), eviction_policy=policy)
    cache.mark_waiting(key, now=dt(0), wait_kind="sleep", resource_id="s")

    cache.mark_active(key, now=dt(1))
    cache.mark_active(key, now=dt(2))  # extra call

    rec = cache.get(key)
    assert rec is not None
    assert rec._wait_count == 0
    assert not rec.is_waiting
