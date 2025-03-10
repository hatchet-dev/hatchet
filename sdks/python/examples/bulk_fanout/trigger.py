import asyncio

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.events import PushEventOptions

hatchet = Hatchet()


async def main() -> None:
    hatchet.event.push(
        "parent:create",
        {"n": 999},
        PushEventOptions(additional_metadata={"no-dedupe": "world"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
