from __future__ import annotations

import asyncio
from datetime import timedelta
from unittest.mock import AsyncMock, MagicMock

from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.worker.durable_eviction.cache import DurableRunRecord

from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.manager import (
    DurableEvictionConfig,
    DurableEvictionManager,
)


def _make_manager(
    cancel_local: MagicMock | None = None,
) -> tuple[DurableEvictionManager, MagicMock]:
    cancel = cancel_local or MagicMock()
    request_eviction = AsyncMock()

    mgr = DurableEvictionManager(
        durable_slots=10,
        cancel_local=cancel,
        request_eviction_with_ack=request_eviction,
        config=DurableEvictionConfig(check_interval=timedelta(hours=1)),
    )
    return mgr, cancel


def test_handle_server_eviction_cancels_and_unregisters() -> None:
    mgr, cancel = _make_manager()

    key = "run-1/0"
    mgr.register_run(
        key,
        step_run_id="ext-1",
        invocation_count=2,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.mark_waiting(key, wait_kind="sleep", resource_id="s1")

    mgr.handle_server_eviction("ext-1", 2)

    cancel.assert_called_once_with(key)
    assert mgr.cache.get(key) is None


def test_handle_server_eviction_unknown_id_is_noop() -> None:
    mgr, cancel = _make_manager()

    mgr.register_run(
        "run-1/0", step_run_id="ext-1", invocation_count=1, eviction_policy=None
    )

    mgr.handle_server_eviction("no-such-id", 1)

    cancel.assert_not_called()
    assert mgr.cache.get("run-1/0") is not None


def test_handle_server_eviction_only_evicts_matching_run() -> None:
    mgr, cancel = _make_manager()

    mgr.register_run(
        "run-1/0",
        step_run_id="ext-1",
        invocation_count=1,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.register_run(
        "run-2/0",
        step_run_id="ext-2",
        invocation_count=1,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.mark_waiting("run-1/0", wait_kind="sleep", resource_id="s1")
    mgr.mark_waiting("run-2/0", wait_kind="sleep", resource_id="s2")

    mgr.handle_server_eviction("ext-1", 1)

    cancel.assert_called_once_with("run-1/0")
    assert mgr.cache.get("run-1/0") is None
    assert mgr.cache.get("run-2/0") is not None


def test_handle_server_eviction_skips_newer_invocation() -> None:
    mgr, cancel = _make_manager()

    mgr.register_run(
        "run-1/0",
        step_run_id="ext-1",
        invocation_count=3,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.mark_waiting("run-1/0", wait_kind="sleep", resource_id="s1")

    mgr.handle_server_eviction("ext-1", 2)

    cancel.assert_not_called()
    assert mgr.cache.get("run-1/0") is not None


def test_handle_server_eviction_evicts_exact_invocation_match() -> None:
    mgr, cancel = _make_manager()

    mgr.register_run(
        "run-1/0",
        step_run_id="ext-1",
        invocation_count=5,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.mark_waiting("run-1/0", wait_kind="sleep", resource_id="s1")

    mgr.handle_server_eviction("ext-1", 5)

    cancel.assert_called_once_with("run-1/0")
    assert mgr.cache.get("run-1/0") is None


def _make_blocking_manager(
    cancel_local: MagicMock,
    ack_started: asyncio.Event,
    release_ack: asyncio.Event,
) -> DurableEvictionManager:
    async def request_eviction(key: ActionKey, rec: DurableRunRecord) -> None:
        ack_started.set()
        await release_ack.wait()

    return DurableEvictionManager(
        durable_slots=10,
        cancel_local=cancel_local,
        request_eviction_with_ack=request_eviction,
        config=DurableEvictionConfig(check_interval=timedelta(hours=1)),
    )


async def test_tick_cancels_locally_when_record_unchanged() -> None:
    cancel = MagicMock()
    ack_started = asyncio.Event()
    release_ack = asyncio.Event()
    release_ack.set()

    mgr = _make_blocking_manager(cancel, ack_started, release_ack)

    key = "run-1/0"
    mgr.register_run(
        key,
        step_run_id="ext-1",
        invocation_count=1,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=0)),
    )
    mgr.mark_waiting(key, wait_kind="wait_for", resource_id="sig-1")

    await asyncio.wait_for(mgr._tick(), timeout=1.0)

    cancel.assert_called_once_with(key)
    assert mgr.cache.get(key) is None


async def test_tick_skips_local_cancel_when_key_reregistered_during_ack() -> None:
    cancel = MagicMock()
    ack_started = asyncio.Event()
    release_ack = asyncio.Event()

    mgr = _make_blocking_manager(cancel, ack_started, release_ack)

    key = "run-1/0"
    mgr.register_run(
        key,
        step_run_id="ext-1",
        invocation_count=1,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=0)),
    )
    mgr.mark_waiting(key, wait_kind="wait_for", resource_id="sig-1")

    tick = asyncio.create_task(mgr._tick())
    await asyncio.wait_for(ack_started.wait(), timeout=1.0)

    mgr.register_run(
        key,
        step_run_id="ext-1",
        invocation_count=2,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )

    release_ack.set()
    await asyncio.wait_for(tick, timeout=1.0)

    cancel.assert_not_called()
    rec = mgr.cache.get(key)
    assert rec is not None
    assert rec.invocation_count == 2


async def test_evict_all_waiting_skips_replaced_record() -> None:
    cancel = MagicMock()

    async def request_eviction(key: ActionKey, rec: DurableRunRecord) -> None:
        mgr.register_run(
            key,
            step_run_id=rec.step_run_id,
            invocation_count=rec.invocation_count + 1,
            eviction_policy=None,
        )

    mgr = DurableEvictionManager(
        durable_slots=10,
        cancel_local=cancel,
        request_eviction_with_ack=request_eviction,
        config=DurableEvictionConfig(check_interval=timedelta(hours=1)),
    )

    key = "run-1/0"
    mgr.register_run(
        key,
        step_run_id="ext-1",
        invocation_count=1,
        eviction_policy=EvictionPolicy(ttl=timedelta(seconds=30)),
    )
    mgr.mark_waiting(key, wait_kind="wait_for", resource_id="sig-1")

    await mgr.evict_all_waiting()

    cancel.assert_not_called()
    rec = mgr.cache.get(key)
    assert rec is not None
    assert rec.invocation_count == 2
