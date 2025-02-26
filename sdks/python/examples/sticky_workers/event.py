from hatchet_sdk.clients.events import PushEventOptions
from hatchet_sdk.hatchet import Hatchet

hatchet = Hatchet(debug=True)

# client.event.push("user:create", {"test": "test"})
hatchet.event.push(
    "sticky:parent",
    {"test": "test"},
    options=PushEventOptions(additional_metadata={"hello": "moon"}),
)
