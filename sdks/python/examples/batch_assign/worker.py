import asyncio
import uuid
from datetime import timedelta
from typing import Any, Never

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.types import BatchMemberId

hatchet = Hatchet()


class OrderedInput(BaseModel):
    index: int


class SimpleInput(BaseModel):
    message: str


class KeyedInput(BaseModel):
    message: str
    group: str


class KeyedFailableInput(BaseModel):
    message: str
    group: str | int


class LargePayloadInput(BaseModel):
    data: str


class BroadcastOutput(BaseModel):
    sum: int


class ChildBatchOutput(BaseModel):
    out: dict[BatchMemberId, SimpleInput]


@hatchet.batch_task(
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=200),
    input_validator=SimpleInput,
)
async def batch_simple(
    tasks: dict[BatchMemberId, SimpleInput], context: Context
) -> dict[BatchMemberId, Any]:
    return {
        id: {"TransformedMessage": inp.message.upper()} for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=2,
    batch_max_interval=timedelta(milliseconds=200),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed(
    tasks: dict[BatchMemberId, KeyedInput], context: Context
) -> dict[BatchMemberId, Any]:
    unique_keys = len({inp.group for _, inp in tasks.items()})
    return {
        id: {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "uppercase": inp.message.upper(),
        }
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=2,
    batch_max_interval=timedelta(milliseconds=200),
    batch_group_key="input.group",
    input_validator=KeyedFailableInput,
)
async def batch_keyed_failable(
    tasks: dict[BatchMemberId, KeyedFailableInput], context: Context
) -> dict[BatchMemberId, Any]:
    return {id: {"uppercase": inp.message.upper()} for id, inp in tasks.items()}


@hatchet.batch_task(
    batch_max_size=3,
    batch_max_interval=timedelta(milliseconds=150),
    batch_group_key="input.group",
    input_validator=KeyedInput,
)
async def batch_keyed_interval(
    tasks: dict[BatchMemberId, KeyedInput],
    context: Context,
) -> dict[BatchMemberId, Any]:
    unique_keys = len({inp.group for _, inp in tasks.items()})
    return {
        id: {
            "batchKey": inp.group,
            "batchSize": len(tasks),
            "uniqueKeys": unique_keys,
            "payload": inp.message,
        }
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=100,
    batch_max_interval=timedelta(seconds=10),
    input_validator=LargePayloadInput,
)
async def batch_large(
    tasks: dict[BatchMemberId, LargePayloadInput], context: Context
) -> dict[BatchMemberId, Any]:
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
    tasks: dict[BatchMemberId, SimpleInput], context: Context
) -> dict[BatchMemberId, Any]:
    return {
        id: {"original": inp.message, "batchSize": len(tasks)}
        for id, inp in tasks.items()
    }


@hatchet.batch_task(
    batch_max_size=20,
    batch_max_interval=timedelta(seconds=2),
    input_validator=OrderedInput,
)
async def batch_ordered(
    tasks: dict[BatchMemberId, OrderedInput], context: Context
) -> dict[BatchMemberId, Any]:
    return {id: {"index": inp.index} for id, inp in tasks.items()}


@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=2),
    input_validator=SimpleInput,
    broadcast_output=True,
)
async def batch_broadcast(
    tasks: dict[BatchMemberId, SimpleInput], context: Context
) -> BroadcastOutput:
    return BroadcastOutput(sum=sum(len(i.message) for _, i in tasks.items()))


@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=2),
    input_validator=SimpleInput,
    broadcast_output=True,
)
async def batch_cancel(_: dict[BatchMemberId, SimpleInput], context: Context) -> None:
    await context.aio_cancel()
    return None


@hatchet.task(input_validator=SimpleInput)
async def child(input: SimpleInput, context: Context) -> dict[str, Any]:
    return {"blahblah": len(input.message)}


@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=60),
    input_validator=SimpleInput,
    broadcast_output=True,
)
async def child_batch(
    inp: dict[BatchMemberId, SimpleInput], context: Context
) -> ChildBatchOutput:
    return ChildBatchOutput(out=inp)


@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=60),
    input_validator=SimpleInput,
    broadcast_output=False,
    execution_timeout=timedelta(seconds=60),
)
async def batch_child_spawn(
    inp: dict[BatchMemberId, SimpleInput], context: Context
) -> dict[BatchMemberId, Any]:
    return {
        id: await child.aio_run(SimpleInput(message="blahblah"))
        for id, inp in inp.items()
    }


@hatchet.batch_task(
    batch_max_size=10,
    batch_max_interval=timedelta(seconds=60),
    input_validator=SimpleInput,
    broadcast_output=False,
    execution_timeout=timedelta(seconds=60),
)
async def batch_child_batch_spawn(
    inp: dict[BatchMemberId, SimpleInput], context: Context
) -> dict[BatchMemberId, Any]:
    async def inner(id: str, inp: SimpleInput) -> Any:
        return id, await child_batch.aio_run(inp)

    ret = {
        id: result
        for id, result in await asyncio.gather(
            *(inner(id, inp) for id, inp in inp.items())
        )
    }
    return ret


def main() -> None:
    worker = hatchet.worker(
        "batch-e2e-worker",
        workflows=[
            batch_simple,
            batch_keyed,
            batch_keyed_failable,
            batch_keyed_interval,
            batch_large,
            batch_single,
            batch_ordered,
            batch_broadcast,
            batch_cancel,
            batch_child_spawn,
            batch_child_batch_spawn,
            child_batch,
            child,
        ],
        slots=25,
    )
    worker.start()


if __name__ == "__main__":
    main()
