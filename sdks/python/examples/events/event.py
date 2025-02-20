from hatchet_sdk import Hatchet

hatchet = Hatchet()
hatchet.event.push("user:create", {"test": "test"})
