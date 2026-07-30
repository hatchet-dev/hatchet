"""Pre-eviction fallback implementations for DurableContext.

These methods support engines older than MIN_DURABLE_EVICTION_VERSION.
Remove this module when support for those engines is dropped.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from hatchet_sdk.clients.listeners.legacy.pre_eviction_durable_event_listener import (
    PreEvictionDurableEventListener,
    RegisterDurableEventRequest,
)
from hatchet_sdk.conditions import (
    OrGroup,
    SleepCondition,
    UserEventCondition,
    flatten_conditions,
)

if TYPE_CHECKING:
    from hatchet_sdk.context.context import DurableContext


async def aio_wait_for_pre_eviction(
    ctx: DurableContext,
    signal_key: str,
    *conditions: SleepCondition | UserEventCondition | OrGroup,
) -> dict[str, Any]:
    if not isinstance(ctx._durable_event_listener, PreEvictionDurableEventListener):
        raise TypeError(
            "Expected PreEvictionDurableEventListener, got "
            f"{type(ctx._durable_event_listener).__name__}"
        )

    task_id = ctx._step_run_id

    request = RegisterDurableEventRequest(
        task_id=task_id,
        signal_key=signal_key,
        conditions=flatten_conditions(list(conditions)),
        config=ctx._runs_client.client_config,
    )

    ctx._durable_event_listener.register_durable_event(request)

    return await ctx._durable_event_listener.result(task_id, signal_key)
