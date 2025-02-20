from dotenv import load_dotenv

from hatchet_sdk.hatchet import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)

hatchet.event.push(
    "affinity:run",
    {"test": "test"},
    options={"additional_metadata": {"hello": "moon"}},
)
