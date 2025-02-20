from hatchet_sdk.clients.events import PushEventOptions
from hatchet_sdk.hatchet import Hatchet

hatchet = Hatchet(debug=True)

hatchet.event.push(
    "affinity:run",
    {"test": "test"},
    options=PushEventOptions(additional_metadata={"hello": "moon"}),
)
