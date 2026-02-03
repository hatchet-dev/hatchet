# > Simple

from hatchet_sdk import Context, EmptyModel, Hatchet
import time

hatchet = Hatchet(debug=True)


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(30)
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(30)
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[simple, simple_durable])
    worker.start()


# !!

if __name__ == "__main__":
    main()
