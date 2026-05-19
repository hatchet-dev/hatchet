import asyncio

from examples.fanout_sync.worker import ParentInput, sync_fanout_parent
from hatchet_sdk import Hatchet

hatchet = Hatchet()


async def main() -> None:
    sync_fanout_parent.run(
        ParentInput(n=2),
        additional_metadata={"hello": "moon"},
    )


if __name__ == "__main__":
    asyncio.run(main())
