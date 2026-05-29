# > Simple
from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, IdempotencyConfig

hatchet = Hatchet()


@hatchet.task(idempotency=IdempotencyConfig(key_expression="input.some_id", ttl=None))
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task()
async def simple_durable(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    # durable tasks should be async
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        workflows=[simple],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
