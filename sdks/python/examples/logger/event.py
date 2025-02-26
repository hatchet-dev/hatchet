from hatchet_sdk import PushEventOptions, new_client

client = new_client()

client.event.push(
    "user:create",
    {"test": "test"},
    options=PushEventOptions(additional_metadata={"hello": "moon"}),
)
