import asyncio
import json
from contextlib import asynccontextmanager
from typing import AsyncGenerator, cast
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.events.worker import (
    EVENT_KEY,
    SECONDARY_KEY,
    WILDCARD_KEY,
    EventWorkflowInput,
    event_workflow,
)
from hatchet_sdk.clients.events import (
    BulkPushEventOptions,
    BulkPushEventWithMetadata,
    PushEventOptions,
)
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.contracts.events_pb2 import Event
from hatchet_sdk.hatchet import Hatchet


class ProcessedEvent(BaseModel):
    id: str
    payload: dict[str, str | bool]
    meta: dict[str, str | bool | int]
    should_have_runs: bool
    test_run_id: str

    def __hash__(self) -> int:
        return hash(self.model_dump_json())


@asynccontextmanager
async def event_filter(
    hatchet: Hatchet,
    test_run_id: str,
    expression: str | None = None,
    payload: dict[str, str] = {},
) -> AsyncGenerator[None, None]:
    expression = (
        expression
        or f"input.should_skip == false && payload.testRunId == '{test_run_id}'"
    )

    f = await hatchet.filters.aio_create(
        workflow_id=event_workflow.id,
        expression=expression,
        scope=test_run_id,
        payload={"testRunId": test_run_id, **payload},
    )

    yield

    await hatchet.filters.aio_delete(f.metadata.id)


async def fetch_runs_for_event(
    hatchet: Hatchet, event: Event
) -> tuple[ProcessedEvent, list[V1TaskSummary]]:
    runs = await hatchet.runs.aio_list(triggering_event_external_id=event.eventId)

    meta = (
        cast(dict[str, str | int | bool], json.loads(event.additionalMetadata))
        if event.additionalMetadata
        else {}
    )
    payload = (
        cast(dict[str, str | bool], json.loads(event.payload)) if event.payload else {}
    )

    return (
        ProcessedEvent(
            id=event.eventId,
            payload=payload,
            meta=meta,
            should_have_runs=meta.get("should_have_runs", False) is True,
            test_run_id=cast(str, meta["test_run_id"]),
        ),
        runs.rows or [],
    )


async def wait_for_result(
    hatchet: Hatchet, events: list[Event]
) -> dict[ProcessedEvent, list[V1TaskSummary]]:
    await asyncio.sleep(3)

    persisted = (await hatchet.event.aio_list(limit=100)).rows or []

    assert {e.eventId for e in events}.issubset({e.metadata.id for e in persisted})

    iters = 0
    while True:
        print("Waiting for event runs to complete...")
        if iters > 15:
            print("Timed out waiting for event runs to complete.")
            return {event.eventId: [] for event in events}

        iters += 1

        event_runs = await asyncio.gather(
            *[fetch_runs_for_event(hatchet, event) for event in events]
        )

        all_empty = all(not event_run for _, event_run in event_runs)

        if all_empty:
            await asyncio.sleep(1)
            continue

        event_id_to_runs = {event_id: runs for (event_id, runs) in event_runs}

        any_queued_or_running = any(
            run.status in [V1TaskStatus.QUEUED, V1TaskStatus.RUNNING]
            for runs in event_id_to_runs.values()
            for run in runs
        )

        if any_queued_or_running:
            await asyncio.sleep(1)
            continue

        break

    return event_id_to_runs


async def assert_event_runs_processed(
    event: ProcessedEvent,
    runs: list[V1TaskSummary],
) -> None:
    if event.should_have_runs:
        assert len(runs) > 0
    else:
        assert len(runs) == 0


def bpi(
    index: int = 1,
    test_run_id: str = "",
    should_skip: bool = False,
    should_have_runs: bool = True,
    key: str = EVENT_KEY,
    payload: dict[str, str] = {},
    scope: str | None = None,
) -> BulkPushEventWithMetadata:
    return BulkPushEventWithMetadata(
        key=key,
        payload={
            "should_skip": should_skip,
            **payload,
        },
        additional_metadata={
            "should_have_runs": should_have_runs,
            "test_run_id": test_run_id,
            "key": index,
        },
        scope=scope,
    )


def cp(should_skip: bool) -> dict[str, bool]:
    return EventWorkflowInput(should_skip=should_skip).model_dump()


@pytest.mark.asyncio(loop_scope="session")
async def test_event_push(hatchet: Hatchet) -> None:
    e = hatchet.event.push(EVENT_KEY, cp(False))

    assert e.eventId is not None


@pytest.mark.asyncio(loop_scope="session")
async def test_async_event_push(hatchet: Hatchet) -> None:
    e = await hatchet.event.aio_push(EVENT_KEY, cp(False))

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


@pytest.fixture(scope="function")
def test_run_id() -> str:
    return str(uuid4())


@pytest.mark.asyncio(loop_scope="session")
async def test_event_engine_behavior(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    events = [
        bpi(
            test_run_id=test_run_id,
        ),
        bpi(
            test_run_id=test_run_id,
            key="thisisafakeeventfoobarbaz",
            should_have_runs=False,
        ),
    ]

    print("Events:", events)

    result = await hatchet.event.aio_bulk_push(events)

    print("Result:", result)

    runs = await wait_for_result(hatchet, result)

    for event, r in runs.items():
        await assert_event_runs_processed(event, r)


def gen_bulk_events(test_run_id: str) -> list[BulkPushEventWithMetadata]:
    return [
        bpi(
            index=1,
            test_run_id=test_run_id,
            should_skip=False,
            should_have_runs=True,
        ),
        bpi(
            index=2,
            test_run_id=test_run_id,
            should_skip=True,
            should_have_runs=True,
        ),
        bpi(
            index=3,
            test_run_id=test_run_id,
            should_skip=False,
            should_have_runs=True,
            scope=test_run_id,
        ),
        bpi(
            index=4,
            test_run_id=test_run_id,
            should_skip=True,
            should_have_runs=False,
            scope=test_run_id,
        ),
        bpi(
            index=5,
            test_run_id=test_run_id,
            should_skip=True,
            should_have_runs=False,
            scope=test_run_id,
            key="thisisafakeeventfoobarbaz",
        ),
        bpi(
            index=6,
            test_run_id=test_run_id,
            should_skip=False,
            should_have_runs=False,
            scope=test_run_id,
            key="thisisafakeeventfoobarbaz",
        ),
    ]


@pytest.mark.asyncio(loop_scope="session")
async def test_event_skipping_filtering(hatchet: Hatchet, test_run_id: str) -> None:
    async with event_filter(hatchet, test_run_id):
        events = gen_bulk_events(test_run_id)

        result = await hatchet.event.aio_bulk_push(events)

        runs = await wait_for_result(hatchet, result)
        for e, r in runs.items():
            await assert_event_runs_processed(e, r)


async def bulk_to_single(hatchet: Hatchet, event: BulkPushEventWithMetadata) -> Event:
    return await hatchet.event.aio_push(
        event_key=event.key,
        payload=event.payload,
        options=PushEventOptions(
            scope=event.scope,
            additional_metadata=event.additional_metadata,
            priority=event.priority,
        ),
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_event_skipping_filtering_no_bulk(
    hatchet: Hatchet, test_run_id: str
) -> None:
    async with event_filter(hatchet, test_run_id):
        raw_events = gen_bulk_events(test_run_id)
        events = await asyncio.gather(
            *[bulk_to_single(hatchet, event) for event in raw_events]
        )

        result = await wait_for_result(hatchet, events)
        for event, runs in result.items():
            await assert_event_runs_processed(event, runs)


@pytest.mark.asyncio(loop_scope="session")
async def test_event_payload_filtering(hatchet: Hatchet, test_run_id: str) -> None:
    async with event_filter(
        hatchet,
        test_run_id,
        "input.should_skip == false && payload.foobar == 'baz'",
        {"foobar": "qux"},
    ):
        event = await hatchet.event.aio_push(
            event_key=EVENT_KEY,
            payload={"message": "This is event 1", "should_skip": False},
            options=PushEventOptions(
                scope=test_run_id,
                additional_metadata={
                    "should_have_runs": False,
                    "test_run_id": test_run_id,
                    "key": 1,
                },
            ),
        )

        runs = await wait_for_result(hatchet, [event])
        assert len(runs) == 0


@pytest.mark.asyncio(loop_scope="session")
async def test_event_payload_filtering_with_payload_match(
    hatchet: Hatchet, test_run_id: str
) -> None:
    async with event_filter(
        hatchet,
        test_run_id,
        "input.should_skip == false && payload.foobar == 'baz'",
        {"foobar": "baz"},
    ):
        event = await hatchet.event.aio_push(
            event_key=EVENT_KEY,
            payload={"message": "This is event 1", "should_skip": False},
            options=PushEventOptions(
                scope=test_run_id,
                additional_metadata={
                    "should_have_runs": True,
                    "test_run_id": test_run_id,
                    "key": 1,
                },
            ),
        )
        runs = await wait_for_result(hatchet, [event])
        assert len(runs) == 1


@pytest.mark.asyncio(loop_scope="session")
async def test_filtering_by_event_key(hatchet: Hatchet, test_run_id: str) -> None:
    async with event_filter(
        hatchet,
        test_run_id,
        f"event_key == '{SECONDARY_KEY}'",
    ):
        event_1 = await hatchet.event.aio_push(
            event_key=SECONDARY_KEY,
            payload={
                "message": "Should run because filter matches",
                "should_skip": False,
            },
            options=PushEventOptions(
                scope=test_run_id,
                additional_metadata={
                    "should_have_runs": True,
                    "test_run_id": test_run_id,
                },
            ),
        )
        event_2 = await hatchet.event.aio_push(
            event_key=EVENT_KEY,
            payload={
                "message": "Should skip because filter does not match",
                "should_skip": False,
            },
            options=PushEventOptions(
                scope=test_run_id,
                additional_metadata={
                    "should_have_runs": False,
                    "test_run_id": test_run_id,
                },
            ),
        )

        event_to_runs = await wait_for_result(hatchet, [event_1, event_2])

        for event, runs in event_to_runs.items():
            await assert_event_runs_processed(event, runs)


@pytest.mark.asyncio(loop_scope="session")
async def test_key_wildcards(hatchet: Hatchet, test_run_id: str) -> None:
    keys = [
        WILDCARD_KEY.replace("*", "1"),
        WILDCARD_KEY.replace("*", "2"),
        "foobar",
        EVENT_KEY,
    ]

    async with event_filter(
        hatchet,
        test_run_id,
    ):
        events = [
            await hatchet.event.aio_push(
                event_key=key,
                payload={
                    "should_skip": False,
                },
                options=PushEventOptions(
                    scope=test_run_id,
                    additional_metadata={
                        "should_have_runs": key != "foobar",
                        "test_run_id": test_run_id,
                    },
                ),
            )
            for key in keys
        ]

        event_to_runs = await wait_for_result(hatchet, events)

        for event, runs in event_to_runs.items():
            await assert_event_runs_processed(event, runs)
