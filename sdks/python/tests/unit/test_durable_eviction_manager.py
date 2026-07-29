from __future__ import annotations

from datetime import timedelta
from unittest.mock import AsyncMock, MagicMock

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
