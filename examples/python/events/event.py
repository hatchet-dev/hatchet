from hatchet_sdk import Hatchet, PushEventOptions

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
