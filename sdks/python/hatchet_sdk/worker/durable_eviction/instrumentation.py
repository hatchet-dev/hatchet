# TODO-DURABLE: file name
from __future__ import annotations

from collections.abc import AsyncIterator, Iterator
from contextlib import asynccontextmanager, contextmanager

from hatchet_sdk.runnables.action import ActionKey
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_durable_eviction_manager,
    ctx_is_durable,
)
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionManager


def _get_eviction_ctx() -> tuple[ActionKey | None, DurableEvictionManager | None]:
    key = ctx_action_key.get()
    mgr = ctx_durable_eviction_manager.get()

    if not key or mgr is None or not ctx_is_durable.get():
        return None, None

    return key, mgr


@contextmanager
def durable_eviction_wait(wait_kind: str, resource_id: str) -> Iterator[None]:
    """
    Mark an SDK-managed wait for the current durable run (if applicable).

    If not executing inside a durable task run, this is a no-op.
    """

    key, mgr = _get_eviction_ctx()
    if key and mgr is not None:
        mgr.mark_waiting(key, wait_kind=wait_kind, resource_id=resource_id)

    try:
        yield
    finally:
        if key and mgr is not None:
            mgr.mark_active(key)


@asynccontextmanager
async def aio_durable_eviction_wait(
    wait_kind: str, resource_id: str
) -> AsyncIterator[None]:
    """
    Async variant of `durable_eviction_wait`.
    """

    with durable_eviction_wait(wait_kind, resource_id):
        yield
