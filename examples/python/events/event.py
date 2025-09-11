from hatchet_sdk import Hatchet, PushEventOptions
from hatchet_sdk.clients.events import BulkPushEventWithMetadata

hatchet = Hatchet()

# > Event trigger
hatchet.event.push("user:create", {"should_skip": False})

# > Event trigger with metadata
hatchet.event.push(
    "user:create",
    {"userId": "1234", "should_skip": False},
    options=PushEventOptions(
        additional_metadata={"source": "api"}  # Arbitrary key-value pair
    ),
)

# > Bulk event push
hatchet.event.bulk_push(
    events=[
        BulkPushEventWithMetadata(
            key="user:create",
            payload={"userId": str(i), "should_skip": False},
        )
        for i in range(10)
    ]
)
