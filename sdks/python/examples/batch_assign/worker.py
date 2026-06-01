from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


class SimpleInput(BaseModel):
    Message: str


class KeyedInput(BaseModel):
    Message: str
    group: str


class LargePayloadInput(BaseModel):
    data: str


@hatchet.batch_task(
    name="batch-e2e-simple",
    retries=0,
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=200),
    input_validator=SimpleInput,
)
async def batch_simple(tasks: list[tuple[SimpleInput, Context]]) -> list[dict]:
    return [{"TransformedMessage": inp.Message.upper()} for inp, _ctx in tasks]


@hatchet.batch_task(
    name="batch-e2e-keyed",
    retries=0,
    batch_max_size=2,
    batch_max_interval=timedelta(milliseconds=200),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed(tasks: list[tuple[KeyedInput, Context]]) -> list[dict]:
    unique_keys = len({inp.group for inp, _ in tasks})
    return [
        {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "uppercase": inp.Message.upper(),
        }
        for inp, _ctx in tasks
    ]


@hatchet.batch_task(
    name="batch-e2e-keyed-interval",
    retries=0,
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=150),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed_interval(tasks: list[tuple[KeyedInput, Context]]) -> list[dict]:
    unique_keys = len({inp.group for inp, _ in tasks})
    return [
        {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "payload": inp.Message,
        }
        for inp, _ctx in tasks
    ]


@hatchet.batch_task(
    name="batch-e2e-large",
    retries=0,
    batch_max_size=100,
    batch_max_interval=timedelta(seconds=1000),
    input_validator=LargePayloadInput,
)
async def batch_large(tasks: list[tuple[LargePayloadInput, Context]]) -> list[dict]:
    return [
        {
            "received": True,
            "batchSize": len(tasks),
            "dataLength": len(inp.data),
        }
        for inp, _ctx in tasks
    ]


@hatchet.batch_task(
    name="batch-e2e-single",
    retries=0,
    batch_max_size=1,
    batch_max_interval=timedelta(milliseconds=100),
    input_validator=SimpleInput,
)
async def batch_single(tasks: list[tuple[SimpleInput, Context]]) -> list[dict]:
    return [{"original": inp.Message, "batchSize": len(tasks)} for inp, _ctx in tasks]


class OrderedInput(BaseModel):
    index: int


@hatchet.batch_task(
    name="batch-e2e-ordered",
    retries=0,
    batch_max_size=20,
    batch_max_interval=timedelta(seconds=2),
    input_validator=OrderedInput,
)
async def batch_ordered(tasks: list[tuple[OrderedInput, Context]]) -> list[dict]:
    return [{"index": inp.index} for inp, _ctx in tasks]


def main() -> None:
    worker = hatchet.worker(
        "batch-e2e-worker",
        workflows=[batch_simple, batch_keyed, batch_keyed_interval, batch_large, batch_single, batch_ordered],
        slots=25,
    )
    worker.start()


if __name__ == "__main__":
    main()
