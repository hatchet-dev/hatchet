from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionManager


@asynccontextmanager
async def aio_durable_eviction_wait(
    wait_kind: str,
    resource_id: str,
    action_key: ActionKey | None = None,
    eviction_manager: DurableEvictionManager | None = None,
) -> AsyncIterator[None]:
    """
    Mark an SDK-managed wait for the current durable run (if applicable).

    If action_key or eviction_manager is None, this is a no-op.
    """

    if action_key and eviction_manager is not None:
        eviction_manager.mark_waiting(
            action_key, wait_kind=wait_kind, resource_id=resource_id
        )

    try:
        yield
    finally:
        if action_key and eviction_manager is not None:
            eviction_manager.mark_active(action_key)
