import asyncio

from hatchet_sdk import Hatchet, TriggerWorkflowOptions

hatchet = Hatchet()


async def main() -> None:
    hatchet.admin.run_workflow(
        "SyncFanoutParent",
        {"test": "test"},
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
