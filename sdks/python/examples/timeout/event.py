from hatchet_sdk import Hatchet

client = Hatchet()

client.event.push("user:create", {"test": "test"})
