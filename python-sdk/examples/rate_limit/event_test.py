from dotenv import load_dotenv
from hatchet_sdk.hatchet import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)

hatchet.client.event.push(
    "rate_limit:create",
    {
        "test": "test"
    }
)
hatchet.client.event.push(
    "rate_limit:create",
    {
        "test": "test"
    }
)
hatchet.client.event.push(
    "rate_limit:create",
    {
        "test": "test"
    }
)