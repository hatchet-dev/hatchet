# > Simple
from hatchet_sdk import Context, DurableContext, Hatchet

hatchet = Hatchet()


@hatchet.task()
def simple(input: None, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task()
async def simple_durable(input: None, ctx: DurableContext) -> dict[str, str]:
    # durable tasks should be async
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[simple, simple_durable],
    )
    worker.start()



if __name__ == "__main__":
    main()
