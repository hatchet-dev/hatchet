from hatchet_sdk import Hatchet, EmptyModel, Context

hatchet = Hatchet()


@hatchet.task()
async def pausable_workflow(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}
