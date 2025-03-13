import asyncio

from examples.fanout_sync.worker import ParentInput, parent
from hatchet_sdk import Hatchet, TriggerWorkflowOptions

hatchet = Hatchet()


async def main() -> None:
    parent.run(
        ParentInput(n=2),
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
