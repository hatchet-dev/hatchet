import asyncio
from uuid import uuid4

import pytest

from hatchet_sdk.clients.events import BulkPushEventOptions, BulkPushEventWithMetadata
from hatchet_sdk.hatchet import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_event_push(hatchet: Hatchet) -> None:
    e = hatchet.event.push("user:create", {"should_skip": False})

    assert e.eventId is not None


@pytest.mark.asyncio(loop_scope="session")
async def test_async_event_push(hatchet: Hatchet) -> None:
    e = await hatchet.event.aio_push("user:create", {"should_skip": False})

    assert e.eventId is not None


@pytest.mark.asyncio(loop_scope="session")
async def test_async_event_bulk_push(hatchet: Hatchet) -> None:
    events = [
        BulkPushEventWithMetadata(
            key="event1",
            payload={"message": "This is event 1", "should_skip": False},
            additional_metadata={"source": "test", "user_id": "user123"},
        ),
        BulkPushEventWithMetadata(
            key="event2",
            payload={"message": "This is event 2", "should_skip": False},
            additional_metadata={"source": "test", "user_id": "user456"},
        ),
        BulkPushEventWithMetadata(
            key="event3",
            payload={"message": "This is event 3", "should_skip": False},
            additional_metadata={"source": "test", "user_id": "user789"},
        ),
    ]
    opts = BulkPushEventOptions(namespace="bulk-test")

    e = await hatchet.event.aio_bulk_push(events, opts)

    assert len(e) == 3

    # Sort both lists of events by their key to ensure comparison order
    sorted_events = sorted(events, key=lambda x: x.key)
    sorted_returned_events = sorted(e, key=lambda x: x.key)
    namespace = "bulk-test"

    # Check that the returned events match the original events
    for original_event, returned_event in zip(sorted_events, sorted_returned_events):
        assert returned_event.key == namespace + original_event.key


@pytest.mark.asyncio(loop_scope="session")
async def test_event_engine_behavior(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    events = [
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"message": "This is event 1", "should_skip": False},
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
            },
        ),
        BulkPushEventWithMetadata(
            key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event", "should_skip": False},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
            },
        ),
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"message": "This is event 3", "should_skip": True},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
            },
        ),
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"message": "This is event 3", "should_skip": False},
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
            },
        ),
    ]

    result = await hatchet.event.aio_bulk_push(events)

    assert len(result) == len(events)

    await asyncio.sleep(5)

    persisted = await hatchet.event.aio_list(limit=100)

    # assert {e.eventId for e in result}.issubset({e.metadata.id for e in persisted.rows})

    for event in persisted.rows or []:
        meta = event.additional_metadata or {}
        if meta.get("test_run_id") != test_run_id:
            continue

        should_have_runs = meta.get("should_have_runs")

        runs = (await hatchet.runs.aio_list(triggering_event_id=event.metadata.id)).rows

        print(event, runs, "\n")

        if should_have_runs:
            assert len(runs) > 0
        else:
            assert len(runs) == 0

    assert False
