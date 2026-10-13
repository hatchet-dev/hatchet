import asyncio

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


class DagReplayTwiceInput(BaseModel):
    second_duration_seconds: float = 1


dag_replay_twice = hatchet.workflow(
    name="dag-replay-twice", input_validator=DagReplayTwiceInput
)


@dag_replay_twice.task()
async def first(input: DagReplayTwiceInput, ctx: Context) -> dict[str, int]:
    await asyncio.sleep(1)

    if ctx.retry_count == 0:
        raise ValueError("fails on the first attempt and succeeds when replayed")

    return {"retry_count": ctx.retry_count}


@dag_replay_twice.task(parents=[first])
async def second(input: DagReplayTwiceInput, ctx: Context) -> dict[str, int]:
    await asyncio.sleep(input.second_duration_seconds)

    return {"retry_count": ctx.retry_count}


def main() -> None:
    worker = hatchet.worker("dag-replay-twice-worker", workflows=[dag_replay_twice])
    worker.start()


if __name__ == "__main__":
    main()
