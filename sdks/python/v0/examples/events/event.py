from dotenv import load_dotenv

from hatchet_sdk import Hatchet

load_dotenv()

hatchet = Hatchet()
hatchet.event.push("user:create", {"test": "test"})
