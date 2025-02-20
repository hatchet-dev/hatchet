from dotenv import load_dotenv

from hatchet_sdk.hatchet import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)

hatchet.event.push("rate_limit:create", {"test": "1"})
hatchet.event.push("rate_limit:create", {"test": "2"})
hatchet.event.push("rate_limit:create", {"test": "3"})
