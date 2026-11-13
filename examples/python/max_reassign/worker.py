import asyncio
import os
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

FIRST_WORKER_NAME = "max-reassign-worker-a"


class WorkerNameOutput(BaseModel):
    worker_name: str


async def wait_forever_on_first_worker() -> WorkerNameOutput:
    worker_name = os.environ["HATCHET_TEST_WORKER_NAME"]

    if worker_name == FIRST_WORKER_NAME:
        await asyncio.sleep(timedelta(minutes=10).total_seconds())

    return WorkerNameOutput(worker_name=worker_name)


@hatchet.task(execution_timeout=timedelta(minutes=3))
async def reassignable_task(input: EmptyModel, ctx: Context) -> WorkerNameOutput:
    return await wait_forever_on_first_worker()


# > Disallowing reassignment
@hatchet.task(execution_timeout=timedelta(minutes=3), max_reassignments_allowed=0)
async def non_reassignable_task(input: EmptyModel, ctx: Context) -> WorkerNameOutput:
    return await wait_forever_on_first_worker()




def main() -> None:
    worker = hatchet.worker(
        name=os.environ["HATCHET_TEST_WORKER_NAME"],
        workflows=[reassignable_task, non_reassignable_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
