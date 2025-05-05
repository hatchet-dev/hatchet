from hatchet_sdk import Hatchet

hatchet = Hatchet()

# > Event trigger
hatchet.event.push("user:create", {})
