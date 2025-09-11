import random

from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)

# Create a list of events with desired distribution
events = ["1"] * 10000 + ["0"] * 100
random.shuffle(events)

# Send the shuffled events
for group in events:
    hatchet.event.push("concurrency-test", {"group": group})
