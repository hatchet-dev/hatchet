from hatchet_sdk import new_client

client = new_client()

client.event.push("user:create", {"test": "test"})
