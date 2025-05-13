import asyncio
from typing import AsyncGenerator
from uuid import uuid4

import pytest
import pytest_asyncio

from examples.events.worker import event_workflow
from hatchet_sdk.clients.events import (
    BulkPushEventOptions,
    BulkPushEventWithMetadata,
    PushEventOptions,
)
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
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
            payload={"message": "This is event 1"},
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
            },
        ),
        BulkPushEventWithMetadata(
            key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event"},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
            },
        ),
    ]

    result = await hatchet.event.aio_bulk_push(events)

    assert len(result) == len(events)

    await asyncio.sleep(5)

    persisted = (await hatchet.event.aio_list(limit=100)).rows or []

    assert {e.eventId for e in result}.issubset({e.metadata.id for e in persisted})

    for event in persisted or []:
        meta = event.additional_metadata or {}
        if meta.get("test_run_id") != test_run_id:
            continue

        should_have_runs = meta.get("should_have_runs")

        runs = (
            await hatchet.runs.aio_list(triggering_event_external_id=event.metadata.id)
        ).rows

        if should_have_runs:
            assert len(runs) > 0
        else:
            assert len(runs) == 0


@pytest_asyncio.fixture(scope="function", loop_scope="session")
async def filter_fixture(hatchet: Hatchet) -> AsyncGenerator[str, None]:
    test_run_id = str(uuid4())
    filter = await hatchet.filters.aio_create(
        workflow_id=event_workflow.id,
        expression=f"input.should_skip == true && payload.testRunId == '{test_run_id}'",
        resource_hint=test_run_id,
        payload={
            "testRunId": test_run_id,
        },
    )

    yield test_run_id

    await hatchet.filters.aio_delete(filter.metadata.id)


@pytest.mark.asyncio(loop_scope="session")
async def test_event_skipping_filtering(hatchet: Hatchet, filter_fixture: str) -> None:
    test_run_id = filter_fixture
    events = [
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"message": "This is event 1", "should_skip": False},
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
                "key": 1,
            },
            resource_hint=test_run_id,
        ),
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"message": "This is event 2", "should_skip": True},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
                "key": 2,
            },
            resource_hint=test_run_id,
        ),
        BulkPushEventWithMetadata(
            key="user:create",
            payload={
                "message": "This event is missing the resource hint",
                "should_skip": False,
            },
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
                "key": 3,
            },
        ),
        BulkPushEventWithMetadata(
            key="user:create",
            payload={
                "message": "This event is missing the resource hint",
                "should_skip": True,
            },
            additional_metadata={
                "should_have_runs": True,
                "test_run_id": test_run_id,
                "key": 4,
            },
        ),
        BulkPushEventWithMetadata(
            key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event", "should_skip": False},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
                "key": 5,
            },
            resource_hint=test_run_id,
        ),
        BulkPushEventWithMetadata(
            key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event", "should_skip": False},
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
                "key": 6,
            },
            resource_hint=test_run_id,
        ),
    ]

    result = await hatchet.event.aio_bulk_push(events)

    assert len(result) == len(events)

    await asyncio.sleep(10)

    persisted = (await hatchet.event.aio_list(limit=100)).rows or []

    assert {e.eventId for e in result}.issubset({e.metadata.id for e in persisted})

    for event in persisted or []:
        meta = event.additional_metadata or {}
        if meta.get("test_run_id") != test_run_id:
            continue

        should_have_runs = meta.get("should_have_runs")

        runs = (
            await hatchet.runs.aio_list(triggering_event_external_id=event.metadata.id)
        ).rows

        if should_have_runs:
            assert len(runs) > 0
        else:
            assert len(runs) == 0


@pytest.mark.asyncio(loop_scope="session")
async def test_event_skipping_filtering_no_bulk(
    hatchet: Hatchet, filter_fixture: str
) -> None:
    test_run_id = filter_fixture

    tasks = [
        hatchet.event.aio_push(
            event_key="user:create",
            payload={"message": "This is event 1", "should_skip": False},
            options=PushEventOptions(
                resource_hint=test_run_id,
                additional_metadata={
                    "should_have_runs": True,
                    "test_run_id": test_run_id,
                    "key": 1,
                },
            ),
        ),
        hatchet.event.aio_push(
            event_key="user:create",
            payload={"message": "This is event 2", "should_skip": True},
            options=PushEventOptions(
                resource_hint=test_run_id,
                additional_metadata={
                    "should_have_runs": False,
                    "test_run_id": test_run_id,
                    "key": 2,
                },
            ),
        ),
        hatchet.event.aio_push(
            event_key="user:create",
            payload={
                "message": "This event is missing the resource hint",
                "should_skip": False,
            },
            options=PushEventOptions(
                additional_metadata={
                    "should_have_runs": True,
                    "test_run_id": test_run_id,
                    "key": 3,
                },
            ),
        ),
        hatchet.event.aio_push(
            event_key="user:create",
            payload={
                "message": "This event is missing the resource hint",
                "should_skip": True,
            },
            options=PushEventOptions(
                additional_metadata={
                    "should_have_runs": True,
                    "test_run_id": test_run_id,
                    "key": 4,
                },
            ),
        ),
        hatchet.event.aio_push(
            event_key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event", "should_skip": False},
            options=PushEventOptions(
                resource_hint=test_run_id,
                additional_metadata={
                    "should_have_runs": False,
                    "test_run_id": test_run_id,
                    "key": 5,
                },
            ),
        ),
        hatchet.event.aio_push(
            event_key="thisisafakeeventfoobarbaz",
            payload={"message": "This is a fake event", "should_skip": False},
            options=PushEventOptions(
                resource_hint=test_run_id,
                additional_metadata={
                    "should_have_runs": False,
                    "test_run_id": test_run_id,
                    "key": 6,
                },
            ),
        ),
    ]

    result = await asyncio.gather(*tasks)

    assert len(result) == len(tasks)

    await asyncio.sleep(10)

    persisted = (await hatchet.event.aio_list(limit=100)).rows or []

    assert {e.eventId for e in result}.issubset({e.metadata.id for e in persisted})

    for event in persisted or []:
        meta = event.additional_metadata or {}
        if meta.get("test_run_id") != test_run_id:
            continue

        should_have_runs = meta.get("should_have_runs")

        runs = (
            await hatchet.runs.aio_list(triggering_event_external_id=event.metadata.id)
        ).rows

        if should_have_runs:
            assert len(runs) > 0
        else:
            assert len(runs) == 0


@pytest_asyncio.fixture(scope="function", loop_scope="session")
async def filter_with_no_payload_match(hatchet: Hatchet) -> AsyncGenerator[str, None]:
    test_run_id = str(uuid4())
    filter = await hatchet.filters.aio_create(
        workflow_id=event_workflow.id,
        expression="input.should_skip == true && payload.foobar == 'baz'",
        resource_hint=test_run_id,
        payload={
            "testRunId": test_run_id,
            "foobar": "qux",
        },
    )

    yield test_run_id

    await hatchet.filters.aio_delete(filter.metadata.id)


@pytest.mark.asyncio(loop_scope="session")
async def test_event_payload_filtering(
    hatchet: Hatchet, filter_with_no_payload_match: str
) -> None:
    test_run_id = filter_with_no_payload_match
    event = await hatchet.event.aio_push(
        event_key="user:create",
        payload={"message": "This is event 1", "should_skip": True},
        options=PushEventOptions(
            resource_hint=test_run_id,
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
                "key": 1,
            },
        ),
    )

    while True:
        runs = await hatchet.runs.aio_list(triggering_event_external_id=event.eventId)

        if not runs.rows:
            await asyncio.sleep(1)
            continue

        rows = runs.rows

        assert len(rows) == 1

        run = rows[0]

        if run.status in [V1TaskStatus.QUEUED, V1TaskStatus.RUNNING]:
            await asyncio.sleep(1)
            continue

        break

    assert run.status == V1TaskStatus.COMPLETED


@pytest_asyncio.fixture(scope="function", loop_scope="session")
async def filter_with_payload_match(hatchet: Hatchet) -> AsyncGenerator[str, None]:
    test_run_id = str(uuid4())
    filter = await hatchet.filters.aio_create(
        workflow_id=event_workflow.id,
        expression="input.should_skip == true && payload.foobar == 'baz'",
        resource_hint=test_run_id,
        payload={
            "testRunId": test_run_id,
            "foobar": "baz",
        },
    )

    yield test_run_id

    await hatchet.filters.aio_delete(filter.metadata.id)


@pytest.mark.asyncio(loop_scope="session")
async def test_event_payload_filtering_with_payload_match(
    hatchet: Hatchet, filter_with_payload_match: str
) -> None:
    test_run_id = filter_with_payload_match
    event = await hatchet.event.aio_push(
        event_key="user:create",
        payload={"message": "This is event 1", "should_skip": True},
        options=PushEventOptions(
            resource_hint=test_run_id,
            additional_metadata={
                "should_have_runs": False,
                "test_run_id": test_run_id,
                "key": 1,
            },
        ),
    )

    await asyncio.sleep(5)

    runs = await hatchet.runs.aio_list(triggering_event_external_id=event.eventId)

    assert not runs.rows
