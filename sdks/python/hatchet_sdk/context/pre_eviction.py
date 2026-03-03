"""Pre-eviction fallback implementations for DurableContext.

These methods support engines older than MIN_DURABLE_EVICTION_VERSION.
Remove this module when support for those engines is dropped.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, cast

from hatchet_sdk.clients.admin import TriggerWorkflowOptions
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
from hatchet_sdk.runnables.types import EmptyModel, TWorkflowInput

if TYPE_CHECKING:
    from hatchet_sdk.context.context import DurableContext
    from hatchet_sdk.runnables.workflow import BaseWorkflow


async def aio_wait_for_pre_eviction(
    ctx: DurableContext,
    signal_key: str,
    *conditions: SleepCondition | UserEventCondition | OrGroup,
) -> dict[str, Any]:
    assert isinstance(ctx.durable_event_listener, PreEvictionDurableEventListener)

    task_id = ctx.step_run_id

    request = RegisterDurableEventRequest(
        task_id=task_id,
        signal_key=signal_key,
        conditions=flatten_conditions(list(conditions)),
        config=ctx.runs_client.client_config,
    )

    ctx.durable_event_listener.register_durable_event(request)

    result: dict[str, Any] = await ctx.durable_event_listener.result(
        task_id, signal_key
    )
    return result


async def spawn_child_pre_eviction(
    ctx: DurableContext,
    workflow: BaseWorkflow[TWorkflowInput],
    input: TWorkflowInput = cast(Any, EmptyModel()),
    options: TriggerWorkflowOptions | None = None,
) -> dict[str, Any]:
    """Fall back to a non-durable admin-client trigger on old engines."""
    serialized = workflow._serialize_input(input, target="string")
    ref = await ctx.admin_client.aio_run_workflow(
        workflow.config.name,
        serialized,
        options or TriggerWorkflowOptions(),
    )
    return await ref.aio_result()
