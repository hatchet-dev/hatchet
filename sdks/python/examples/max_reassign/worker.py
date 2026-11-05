import argparse
from hatchet_sdk import Hatchet, Context, EmptyModel
import asyncio
from pydantic import BaseModel

hatchet = Hatchet()


class WorkerIdEnvelope(BaseModel):
    worker_id: str


@hatchet.task()
async def worker_id_returner(_i: EmptyModel, ctx: Context) -> WorkerIdEnvelope:
    return WorkerIdEnvelope(worker_id=ctx.worker_id)


@hatchet.task(max_reassignments_allowed=0)
async def no_reassigns_allowed(_i: EmptyModel, ctx: Context) -> None:
    await asyncio.sleep(3)

    await ctx.aio_put_stream(
        WorkerIdEnvelope(worker_id=ctx.worker_id).model_dump_json()
    )

    for _ in range(10):
        await asyncio.sleep(1)

    return None


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--label", type=str, required=True)
    args = parser.parse_args()

    worker = hatchet.worker(
        name="max_reassign_worker",
        workflows=[no_reassigns_allowed, worker_id_returner],
        labels={"no-reassign": args.label},
    )
    worker.start()


if __name__ == "__main__":
    main()
