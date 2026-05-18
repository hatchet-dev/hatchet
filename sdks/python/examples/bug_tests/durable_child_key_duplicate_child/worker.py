# > Simple
from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet

hatchet = Hatchet()


@hatchet.task()
async def child_child_key_bug(_i: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"id": ctx.workflow_run_id}


@hatchet.durable_task()
async def durable_parent_child_key_bug(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    for _ in range(2):
        await child_child_key_bug.aio_run(
            child_key=ctx.workflow_run_id + f"-child",
        )

    return {"result": "Hello, world!"}
