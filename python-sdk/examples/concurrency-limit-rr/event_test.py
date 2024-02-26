from hatchet_sdk import new_client
from dotenv import load_dotenv

load_dotenv()

client = new_client()

for i in range(20):
    group = "0"

    if i > 10:
        group = "1"

    client.event.push(
        "concurrency-test",
        {
            "group": group
        }
    )