import asyncio

from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet()


async def main() -> None:
    workflow_run_refs = bulk_parent_wf.run_many(
        workflows=[
            bulk_parent_wf.create_run_workflow_config(
                input=ParentInput(n=i),
                options=TriggerWorkflowOptions(
                    additional_metadata={
                        "bulk-trigger": i,
                        "hello-{i}": "earth-{i}",
                    }
                ),
            )
            for i in range(20)
        ],
    )

    results = await asyncio.gather(
        *[ref.aio_result() for ref in workflow_run_refs],
        return_exceptions=True,
    )

    for result in results:
        if isinstance(result, Exception):
            print(f"An error occurred: {result}")  # Handle the exception here
        else:
            print(result)


if __name__ == "__main__":
    asyncio.run(main())
