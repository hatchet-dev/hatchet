import asyncio

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet()


async def main() -> None:
    hatchet.admin.run_workflow(
        "Parent",
        {"test": "test"},
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
