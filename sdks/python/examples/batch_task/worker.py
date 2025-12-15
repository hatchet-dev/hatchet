import asyncio

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class BatchInput(BaseModel):
    message: str
    group: str


hatchet = Hatchet(debug=True)

workflow = hatchet.workflow(name="batch-task-example", input_validator=BatchInput)


@workflow.batch_task(
    name="uppercase",
    batch_size=2,
    flush_interval_ms=500,
    batch_key="input.group",
    max_runs=1,
)
async def uppercase(
    inputs: list[BatchInput], ctxs: list[Context]
) -> list[dict[str, str]]:
    # `inputs` is aligned with `ctxs` by index.
    await asyncio.sleep(10)
    return [
        {"group": inputs[i].group, "uppercase": inputs[i].message.upper()}
        for i in range(len(inputs))
    ]


def main() -> None:
    worker = hatchet.worker("batch-task-worker", workflows=[workflow], slots=10)
    worker.start()


if __name__ == "__main__":
    main()
