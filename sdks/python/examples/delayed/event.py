from hatchet_sdk import new_client

client = new_client()

client.event.push("printer:schedule", {"message": "test"})
