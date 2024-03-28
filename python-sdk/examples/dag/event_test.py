from dotenv import load_dotenv

from hatchet_sdk import new_client

load_dotenv()

client = new_client()

for i in range(10):
    client.event.push("user:create", {"test": "test"})
