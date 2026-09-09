import asyncio
import random

from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import Hatchet


async def main() -> None:
    hatchet = Hatchet()

    # Generate a random stream key to use to track all
    # stream events for this workflow run.

    streamKey = "streamKey"
    streamVal = f"sk-{random.randint(1, 100)}"

    # Specify the stream key as additional metadata
    # when running the workflow.

    # This key gets propagated to all child workflows
    # and can have an arbitrary property name.
    bulk_parent_wf.run(
        input=ParentInput(n=2),
        additional_metadata={streamKey: streamVal},
    )


if __name__ == "__main__":
    asyncio.run(main())
