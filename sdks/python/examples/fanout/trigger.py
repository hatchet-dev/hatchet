import asyncio
from typing import Any

from examples.fanout.worker import ChildInput, ParentInput, child_wf, parent_wf
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet()


async def main() -> None:
    await parent_wf.aio_run(
        ParentInput(n=100),
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
