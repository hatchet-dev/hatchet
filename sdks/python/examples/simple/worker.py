# > Simple
from datetime import timedelta
import time
from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionConfig

hatchet = Hatchet(debug=True)


@hatchet.task()
def simple5(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(10)
    return {"result": "Hello, world!"}


@hatchet.durable_task(
    execution_timeout=timedelta(seconds=60),
    eviction=EvictionPolicy(
        ttl=timedelta(seconds=5),
    ),
)
async def simple_durable5(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    print("\033[34mcheckpoint 1\033[0m")
    await simple5.aio_run()
    print("\033[34mcheckpoint 2\033[0m")
    await simple5.aio_run()
    print("\033[34mcheckpoint 3\033[0m")
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[simple5, simple_durable5],
        durable_eviction_config=DurableEvictionConfig(
            check_interval=timedelta(seconds=1),
            reserve_slots=1,
            min_wait_for_capacity_eviction=timedelta(seconds=0),
        ),
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
