# > Simple
from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionConfig

hatchet = Hatchet(debug=True)


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task()
async def simple_durable(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    # durable tasks should be async
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[simple, simple_durable],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
