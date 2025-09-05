import asyncio

from examples.fanout.worker import ParentInput, parent_wf, child_wf, ChildInput
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from typing import Any

hatchet = Hatchet()


async def main() -> None:
    await parent_wf.aio_run(
        ParentInput(n=2),
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )


# > Bulk run children
async def run_child_workflows(n: int) -> list[dict[str, Any]]:
    return await child_wf.aio_run_many(
        [
            child_wf.create_bulk_run_item(
                input=ChildInput(a=str(i)),
            )
            for i in range(n)
        ]
    )


# !!

if __name__ == "__main__":
    asyncio.run(main())
