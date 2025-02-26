from hatchet_sdk import new_client
from hatchet_sdk.clients.events import (
    BulkPushEventOptions,
    BulkPushEventWithMetadata,
    PushEventOptions,
)

client = new_client()

# client.event.push("user:create", {"test": "test"})
client.event.push(
    "user:create",
    {"test": "test"},
    options=PushEventOptions(additional_metadata={"hello": "moon"}),
)

events = [
    BulkPushEventWithMetadata(
        key="event1",
        payload={"message": "This is event 1"},
        additional_metadata={"source": "test", "user_id": "user123"},
    ),
    BulkPushEventWithMetadata(
        key="event2",
        payload={"message": "This is event 2"},
        additional_metadata={"source": "test", "user_id": "user456"},
    ),
    BulkPushEventWithMetadata(
        key="event3",
        payload={"message": "This is event 3"},
        additional_metadata={"source": "test", "user_id": "user789"},
    ),
]


result = client.event.bulk_push(
    events, options=BulkPushEventOptions(namespace="bulk-test")
)

print(result)
