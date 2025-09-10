# > Simple

from hatchet_sdk import Context, EmptyModel, Hatchet, DefaultFilter

hatchet = Hatchet(debug=True)


@hatchet.task(
    default_filters=[
        DefaultFilter(
            expression="input.foo == 'bar'",
            scope="some-test-scope",
            payload={
                "key": "value",
                "number": 123,
            },
        )
    ]
)
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[simple, simple_durable])
    worker.start()


# !!

if __name__ == "__main__":
    main()
