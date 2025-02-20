from typing import List

import pytest

from hatchet_sdk.clients.events import BulkPushEventOptions, BulkPushEventWithMetadata
from hatchet_sdk.hatchet import Hatchet


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
async def test_event_push(hatchet: Hatchet) -> None:
    e = hatchet.event.push("user:create", {"test": "test"})

    assert e.eventId is not None


@pytest.mark.asyncio(scope="session")
async def test_async_event_push(aiohatchet: Hatchet) -> None:
    e = await aiohatchet.event.async_push("user:create", {"test": "test"})

    assert e.eventId is not None


@pytest.mark.asyncio(scope="session")
async def test_async_event_bulk_push(aiohatchet: Hatchet) -> None:

    events: List[BulkPushEventWithMetadata] = [
        {
            "key": "event1",
            "payload": {"message": "This is event 1"},
            "additional_metadata": {"source": "test", "user_id": "user123"},
        },
        {
            "key": "event2",
            "payload": {"message": "This is event 2"},
            "additional_metadata": {"source": "test", "user_id": "user456"},
        },
        {
            "key": "event3",
            "payload": {"message": "This is event 3"},
            "additional_metadata": {"source": "test", "user_id": "user789"},
        },
    ]
    opts: BulkPushEventOptions = {"namespace": "bulk-test"}

    e = await aiohatchet.event.async_bulk_push(events, opts)

    assert len(e) == 3

    # Sort both lists of events by their key to ensure comparison order
    sorted_events = sorted(events, key=lambda x: x["key"])
    sorted_returned_events = sorted(e, key=lambda x: x.key)
    namespace = "bulk-test"

    # Check that the returned events match the original events
    for original_event, returned_event in zip(sorted_events, sorted_returned_events):
        assert returned_event.key == namespace + original_event["key"]
