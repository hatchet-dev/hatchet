import asyncio

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


# > Define models
class TaskInput(BaseModel):
    user_id: int


class TaskOutput(BaseModel):
    ok: bool




async def main() -> None:
    hatchet = Hatchet()

    # > Create a stub task
    stub = hatchet.stubs.task(
        # make sure the name and schemas exactly match the implementation
        name="externally-triggered-task",
        input_validator=TaskInput,
        output_validator=TaskOutput,
    )

    # > Trigger the task
    # input type checks properly
    result = await stub.aio_run(input=TaskInput(user_id=1234))

    # `result.ok` type checks properly
    print("Is successful:", result.ok)


if __name__ == "__main__":
    asyncio.run(main())
