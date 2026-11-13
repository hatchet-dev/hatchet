import asyncio
import json

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


# > Publishing
class ProgressInput(BaseModel):
    topic: str


@hatchet.task(input_validator=ProgressInput)
async def durable_streams_progress(
    input: ProgressInput, ctx: Context
) -> dict[str, int]:
    total = 5

    for step in range(1, total + 1):
        await asyncio.sleep(0.5)
        await hatchet.streams.aio_publish(
            input.topic, json.dumps({"step": step, "total": total})
        )

    return {"steps": total}




# > Reset on retry
@hatchet.task(input_validator=ProgressInput, retries=1)
async def durable_streams_retrying_progress(
    input: ProgressInput, ctx: Context
) -> dict[str, int]:
    await hatchet.streams.aio_reset(input.topic)

    for chunk in range(3):
        await hatchet.streams.aio_publish(
            input.topic, json.dumps({"attempt": ctx.retry_count, "chunk": chunk})
        )

    if ctx.retry_count == 0:
        raise RuntimeError("intentional failure before the final message")

    await hatchet.streams.aio_publish(input.topic, json.dumps({"done": True}))

    return {"chunks": 3}




def main() -> None:
    worker = hatchet.worker(
        "durable-streams-worker",
        workflows=[durable_streams_progress, durable_streams_retrying_progress],
    )
    worker.start()


if __name__ == "__main__":
    main()
