from typing import List

from dotenv import load_dotenv

from hatchet_sdk import new_client
from hatchet_sdk.clients.events import BulkPushEventWithMetadata

load_dotenv()

client = new_client()

# client.event.push("user:create", {"test": "test"})
client.event.push(
    "user:create", {"test": "test"}, options={"additional_metadata": {"hello": "moon"}}
)

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


result = client.event.bulk_push(
    events,
    options={"namespace": "bulk-test"},
)

print(result)
