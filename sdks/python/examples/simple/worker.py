# > Simple
from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionConfig
from datetime import timedelta
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task(
    execution_timeout=timedelta(seconds=60),
    eviction=EvictionPolicy(
        ttl=timedelta(seconds=5),
    ),
)
async def simple_durable(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    # durable tasks should be async
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[simple, simple_durable],
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
