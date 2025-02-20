from dotenv import load_dotenv

from hatchet_sdk import PushEventOptions, new_client

load_dotenv()

client = new_client()

# client.event.push("user:create", {"test": "test"})
client.event.push(
    "user:create", {"test": "test"}, options={"additional_metadata": {"hello": "moon"}}
)
