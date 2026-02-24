# > Simple
from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


def helper() -> None:
    ctx = hatchet.get_current_context()

    print(ctx)


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    helper()
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
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
