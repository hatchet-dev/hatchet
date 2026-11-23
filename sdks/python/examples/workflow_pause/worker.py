from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


@hatchet.task()
async def pausable_workflow(input: None, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}
