from hatchet_sdk.hatchet import Hatchet

hatchet = Hatchet(debug=True)

hatchet.event.push("rate_limit:create", {"test": "1"})
hatchet.event.push("rate_limit:create", {"test": "2"})
hatchet.event.push("rate_limit:create", {"test": "3"})
