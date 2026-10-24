from collections.abc import AsyncGenerator
from hatchet_sdk import Context, EmptyModel, Hatchet


from hatchet_sdk import Hatchet
import logging
import time
import asyncio

hatchet = Hatchet()

logger = logging.getLogger(__name__)


async def lifespan() -> AsyncGenerator[None, None]:
    import asyncio
    from concurrent.futures import ThreadPoolExecutor

    # the default default (hehe) executor is instantiated with ThreadPoolExecutor()
    # which has max_workers = min(32, (os.cpu_count() or 1) + 4)
    # 5 simulates a single core worker
    asyncio.get_running_loop().set_default_executor(
        ThreadPoolExecutor(max_workers=5, thread_name_prefix="tiny-default-pool")
    )
    yield


def sync_code_i_dont_control() -> None:
    time.sleep(2)


@hatchet.task()
async def my_async_task_that_i_do_control(input: EmptyModel, ctx: Context) -> None:
    await asyncio.to_thread(sync_code_i_dont_control)


def main() -> None:
    worker = hatchet.worker(
        "worker_that_will_be_exhausted",
        slots=50,
        workflows=[
            my_async_task_that_i_do_control,
        ],
        lifespan=lifespan,
    )

    worker.start()


if __name__ == "__main__":
    main()
