import asyncio

import pytest

from examples.batch_assign.worker import (
    KeyedInput,
    LargePayloadInput,
    SimpleInput,
    batch_keyed,
    batch_keyed_interval,
    batch_large,
    batch_simple,
    batch_single,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_flushes_when_batch_size_is_reached() -> None:
    inputs = ["alpha", "bravo", "charlie"]

    results = await asyncio.gather(
        *[batch_simple.aio_run(SimpleInput(Message=msg)) for msg in inputs]
    )

    assert len(results) == 3
    assert [r["TransformedMessage"] for r in results] == [m.upper() for m in inputs]


@pytest.mark.asyncio(loop_scope="session")
async def test_flushes_when_fewer_items_buffered_than_batch_size() -> None:
    inputs = ["delta", "echo"]

    futures = [batch_simple.aio_run(SimpleInput(Message=msg)) for msg in inputs]
    await asyncio.sleep(0.5)
    results = await asyncio.gather(*futures)

    assert [r["TransformedMessage"] for r in results] == [m.upper() for m in inputs]


@pytest.mark.asyncio(loop_scope="session")
async def test_partitions_batches_by_key_when_batch_size_reached() -> None:
    inputs = [
        KeyedInput(Message="alpha", group="tenant-1"),
        KeyedInput(Message="bravo", group="tenant-1"),
        KeyedInput(Message="charlie", group="tenant-2"),
        KeyedInput(Message="delta", group="tenant-2"),
    ]

    results = await asyncio.gather(*[batch_keyed.aio_run(inp) for inp in inputs])

    assert len(results) == len(inputs)
    for result, inp in zip(results, inputs):
        assert result["batchKey"] == inp.group
        assert result["batchSize"] == 2
        assert result["uniqueKeys"] == 1
        assert result["uppercase"] == inp.Message.upper()


@pytest.mark.asyncio(loop_scope="session")
async def test_flushes_keyed_batches_independently_when_interval_elapses() -> None:
    inputs = [
        KeyedInput(Message="echo", group="tenant-1"),
        KeyedInput(Message="foxtrot", group="tenant-1"),
        KeyedInput(Message="golf", group="tenant-1"),
        KeyedInput(Message="hotel", group="tenant-2"),
    ]

    results = await asyncio.gather(*[batch_keyed_interval.aio_run(inp) for inp in inputs])

    assert [r["batchKey"] for r in results] == [inp.group for inp in inputs]
    assert all(r["batchSize"] == 3 for r in results[:3])
    assert results[3]["batchSize"] == 1
    assert all(r["uniqueKeys"] == 1 for r in results)
    assert results[3]["payload"] == "hotel"


@pytest.mark.asyncio(loop_scope="session")
async def test_completes_all_tasks_with_large_payloads() -> None:
    payload = "x" * 4_000_000
    task_count = 100

    results = await asyncio.gather(
        *[batch_large.aio_run(LargePayloadInput(data=payload)) for _ in range(task_count)]
    )

    assert len(results) == task_count
    assert all(r["received"] for r in results)
    assert all(r["dataLength"] == 4_000_000 for r in results)
    assert all(r["batchSize"] == task_count for r in results)


@pytest.mark.asyncio(loop_scope="session")
async def test_handles_batch_size_of_one_without_keys() -> None:
    inputs = ["india", "juliet"]

    results = await asyncio.gather(
        *[batch_single.aio_run(SimpleInput(Message=msg)) for msg in inputs]
    )

    assert [r["batchSize"] for r in results] == [1, 1]
    assert [r["original"] for r in results] == inputs
