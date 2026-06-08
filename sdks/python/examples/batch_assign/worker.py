import uuid
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from typing import Any

hatchet = Hatchet()


class OrderedInput(BaseModel):
    index: int


class SimpleInput(BaseModel):
    Message: str


class KeyedInput(BaseModel):
    Message: str
    group: str


class LargePayloadInput(BaseModel):
    data: str


@hatchet.batch_task(
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=200),
    input_validator=SimpleInput,
)
async def batch_simple(
    tasks: dict[str, SimpleInput],
    context: Context
) -> dict[str, Any]:
    return {id: {"TransformedMessage": inp.Message.upper()} for id, inp in tasks.items()}


@hatchet.batch_task(
    batch_max_size=2,
    batch_max_interval=timedelta(milliseconds=200),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed(tasks: dict[str, KeyedInput], context: Context) -> dict[str, Any]:
    unique_keys = len({inp.group for _, inp in tasks.items()})
    return {
        id:
        {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "uppercase": inp.Message.upper(),
        }
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=150),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed_interval(
    tasks: dict[str, KeyedInput], context: Context,
) -> dict[str, Any]:
    unique_keys = len({inp.group for _, inp in tasks.items()})
    return {
        id:
        {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "payload": inp.Message,
        }
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=100,
    batch_max_interval=timedelta(seconds=1),
    input_validator=LargePayloadInput,
)
async def batch_large(
    tasks: dict[str, LargePayloadInput],
    context: Context
) -> dict[str, Any]:
    batch_id = str(uuid.uuid4())
    return {
        id: {
            "batchId": batch_id,
            "received": True,
            "batchSize": len(tasks),
            "dataLength": len(inp.data),
        }
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=1,
    batch_max_interval=timedelta(milliseconds=100),
    input_validator=SimpleInput,
)
async def batch_single(
    tasks: dict[str, SimpleInput],
    context: Context
) -> dict[str, Any]:
    return { id: {"original": inp.Message, "batchSize": len(tasks)} for id, inp in tasks.items()}


@hatchet.batch_task(
    batch_max_size=20,
    batch_max_interval=timedelta(seconds=2),
    input_validator=OrderedInput,
)
async def batch_ordered(
    tasks: dict[str, OrderedInput],
    context: Context
) -> dict[str, Any]:
    return {id: {"index": inp.index} for id, inp in tasks.items()}

@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=2),
    input_validator=SimpleInput,
    broadcast_output=True,
)
async def batch_broadcast(
        tasks: dict[str, SimpleInput],
        context: Context
) -> dict:
    return {"sum": sum(len(i.Message) for _, i in tasks.items())}


def main() -> None:
    worker = hatchet.worker(
        "batch-e2e-worker",
        workflows=[
            batch_simple,
            batch_keyed,
            batch_keyed_interval,
            batch_large,
            batch_single,
            batch_ordered,
            batch_broadcast,
        ],
        slots=25,
    )
    worker.start()


if __name__ == "__main__":
    main()
