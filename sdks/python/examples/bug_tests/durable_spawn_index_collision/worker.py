from typing import Any, Literal

from pydantic import BaseModel

from hatchet_sdk import Context, DurableContext, Hatchet
from hatchet_sdk.runnables.contextvars import ctx_action_key, workflow_spawn_indices

hatchet = Hatchet()


class SpawnIndexCollisionInput(BaseModel):
    scenario: Literal["collision", "self_dedupe"]


class OutputA(BaseModel):
    which_child: Literal["a"]


class OutputB(BaseModel):
    which_child: Literal["b"]


@hatchet.task()
async def spawn_index_child_a(_i: None, ctx: Context) -> OutputA:
    return OutputA(which_child="a")


@hatchet.task()
async def spawn_index_child_b(_i: None, ctx: Context) -> OutputB:
    return OutputB(which_child="b")


@hatchet.durable_task(input_validator=SpawnIndexCollisionInput)
async def durable_spawn_index_collision(
    input: SpawnIndexCollisionInput, ctx: DurableContext
) -> dict[str, Any]:
    first = await spawn_index_child_a.aio_run()  # child_index 0
    await spawn_index_child_b.aio_run()  # child_index 1

    key = ctx_action_key.get()
    assert key is not None

    if input.scenario == "collision":
        # next spawn claims index 1, which is bound to child B
        workflow_spawn_indices[key] -= 1
    else:
        # next spawn claims index 0, child A's own binding
        workflow_spawn_indices[key] -= 2

    respawned = await spawn_index_child_a.aio_run()

    return {"first": first, "respawned": respawned}
