# > Simple

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.v0 import Hatchet as HatchetV0

hatchet = Hatchet(debug=True)
# hatchet_v0 = HatchetV0(debug=True)


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[simple, simple_durable])
    worker.start()



if __name__ == "__main__":
    main()
