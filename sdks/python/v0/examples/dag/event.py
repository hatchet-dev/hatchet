from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)

# for i in range(10):
hatchet.event.push("dag:create", {"test": "test"})
