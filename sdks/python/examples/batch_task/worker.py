import asyncio
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class BatchInput(BaseModel):
    message: str
    group: str


hatchet = Hatchet(debug=True)

workflow = hatchet.workflow(name="batch-task-example", input_validator=BatchInput)


@workflow.batch_task(
    name="uppercase",
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=500),
    batch_group_key="input.group",
    batch_group_max_runs=1,
)
async def uppercase(
    tasks: list[tuple[BatchInput, Context]]
) -> list[dict[str, str]]:
    # Each item is a tuple of (input, ctx).
    await asyncio.sleep(10)
    return [
        {"group": inp.group, "uppercase": inp.message.upper()} for inp, _ctx in tasks
    ]


def main() -> None:
    worker = hatchet.worker("batch-task-worker", workflows=[workflow], slots=10)
    worker.start()


if __name__ == "__main__":
    main()
