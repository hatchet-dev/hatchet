from hatchet_sdk import Hatchet

hatchet = Hatchet()

for i in range(200):
    group = "0"

    if i % 2 == 0:
        group = "1"

    hatchet.event.push("concurrency-test", {"group": group})
