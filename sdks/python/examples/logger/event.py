from hatchet_sdk import Hatchet, PushEventOptions

hatchet = Hatchet()

hatchet.event.push(
    "user:create",
    {"test": "test"},
    options=PushEventOptions(additional_metadata={"hello": "moon"}),
)
