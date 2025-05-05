import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import pytest\n\nfrom hatchet_sdk.clients.events import BulkPushEventOptions, BulkPushEventWithMetadata\nfrom hatchet_sdk.hatchet import Hatchet\n\n\n@pytest.mark.asyncio(loop_scope='session')\nasync def test_event_push(hatchet: Hatchet) -> None:\n    e = hatchet.event.push('user:create', {'test': 'test'})\n\n    assert e.eventId is not None\n\n\n@pytest.mark.asyncio(loop_scope='session')\nasync def test_async_event_push(hatchet: Hatchet) -> None:\n    e = await hatchet.event.aio_push('user:create', {'test': 'test'})\n\n    assert e.eventId is not None\n\n\n@pytest.mark.asyncio(loop_scope='session')\nasync def test_async_event_bulk_push(hatchet: Hatchet) -> None:\n\n    events = [\n        BulkPushEventWithMetadata(\n            key='event1',\n            payload={'message': 'This is event 1'},\n            additional_metadata={'source': 'test', 'user_id': 'user123'},\n        ),\n        BulkPushEventWithMetadata(\n            key='event2',\n            payload={'message': 'This is event 2'},\n            additional_metadata={'source': 'test', 'user_id': 'user456'},\n        ),\n        BulkPushEventWithMetadata(\n            key='event3',\n            payload={'message': 'This is event 3'},\n            additional_metadata={'source': 'test', 'user_id': 'user789'},\n        ),\n    ]\n    opts = BulkPushEventOptions(namespace='bulk-test')\n\n    e = await hatchet.event.aio_bulk_push(events, opts)\n\n    assert len(e) == 3\n\n    # Sort both lists of events by their key to ensure comparison order\n    sorted_events = sorted(events, key=lambda x: x.key)\n    sorted_returned_events = sorted(e, key=lambda x: x.key)\n    namespace = 'bulk-test'\n\n    # Check that the returned events match the original events\n    for original_event, returned_event in zip(sorted_events, sorted_returned_events):\n        assert returned_event.key == namespace + original_event.key\n",
  source: 'out/python/events/test_event.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
