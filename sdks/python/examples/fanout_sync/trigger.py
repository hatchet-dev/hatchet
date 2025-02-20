import asyncio

from dotenv import load_dotenv

from hatchet_sdk import TriggerWorkflowOptions, new_client


async def main() -> None:
    load_dotenv()
    hatchet = new_client()

    hatchet.admin.run_workflow(
        "SyncFanoutParent",
        {"test": "test"},
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )


if __name__ == "__main__":
    asyncio.run(main())
