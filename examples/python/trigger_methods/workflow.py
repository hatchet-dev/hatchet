from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


# > Define a task
class HelloInput(BaseModel):
    name: str


class HelloOutput(BaseModel):
    greeting: str


@hatchet.task(input_validator=HelloInput)
async def say_hello(input: HelloInput, ctx: Context) -> HelloOutput:
    return HelloOutput(greeting=f"Hello, {input.name}!")




async def main() -> None:
    # > Sync
    ref = say_hello.run(input=HelloInput(name="World"), wait_for_result=False)

    # > Async
    ref = await say_hello.aio_run(
        input=HelloInput(name="Async World"), wait_for_result=False
    )

    # > Result Sync
    result = ref.result()

    # > Result Async
    result = await ref.aio_result()
