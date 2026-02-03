# > Simple

import time

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


@hatchet.task(slot_requirements={"default": 40})
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(30)
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
    time.sleep(30)
    return {"result": "Hello, world!"}


def main() -> None:
    worker = hatchet.worker(
        "test-worker",
        slot_capacities={"default": 100, "durable": 2},
        # TODO: default slot configs
        # slots=3,
        # durable_slots=55,
        workflows=[simple, simple_durable],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
