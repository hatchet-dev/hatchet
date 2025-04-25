import asyncio

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet()

async def main() -> None:
    await parent_wf.aio_run(
        ParentInput(n=2),
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )

if __name__ == "__main__":
    asyncio.run(main())
