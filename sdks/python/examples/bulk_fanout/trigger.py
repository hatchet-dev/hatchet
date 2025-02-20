import asyncio

from hatchet_sdk import new_client
from hatchet_sdk.clients.events import PushEventOptions


async def main() -> None:

    hatchet = new_client()

    hatchet.event.push(
        "parent:create",
        {"n": 999},
        PushEventOptions(additional_metadata={"no-dedupe": "world"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
