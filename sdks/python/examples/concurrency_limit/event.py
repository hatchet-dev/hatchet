from hatchet_sdk import Hatchet

hatchet = Hatchet()

hatchet.event.push("concurrency-test", {"group": "test"})
