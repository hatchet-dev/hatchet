import asyncio

from hatchet_sdk import new_client
from hatchet_sdk.clients.admin import TriggerWorkflowOptions, WorkflowRunDict


async def main() -> None:

    hatchet = new_client()

    workflow_runs = [
        WorkflowRunDict(
            workflow_name="BulkParent",
            input={"n": i},
            options=TriggerWorkflowOptions(
                additional_metadata={
                    "bulk-trigger": i,
                    "hello-{i}": "earth-{i}",
                }
            ),
        )
        for i in range(20)
    ]

    workflowRunRefs = hatchet.admin.run_workflows(
        workflow_runs,
    )

    results = await asyncio.gather(
        *[workflowRunRef.result() for workflowRunRef in workflowRunRefs],
        return_exceptions=True,
    )

    for result in results:
        if isinstance(result, Exception):
            print(f"An error occurred: {result}")  # Handle the exception here
        else:
            print(result)


if __name__ == "__main__":
    asyncio.run(main())
