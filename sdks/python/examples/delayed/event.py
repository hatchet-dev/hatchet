from hatchet_sdk import Hatchet

hatchet = Hatchet()

hatchet.event.push("printer:schedule", {"message": "test"})
