from dotenv import load_dotenv

from hatchet_sdk import new_client

load_dotenv()

client = new_client()

for i in range(200):
    group = "0"

    if i % 2 == 0:
        group = "1"

    client.event.push("concurrency-test", {"group": group})
