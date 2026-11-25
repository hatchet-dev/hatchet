import asyncio
import random

from examples.fanout.worker import ParentInput, parent_wf


async def main() -> None:
    # Generate a random stream key to use to track all
    # stream events for this workflow run.

    stream_key = "streamKey"
    stream_value = f"sk-{random.randint(1, 100)}"

    # Specify the stream key as additional metadata
    # when running the workflow.

    # This key gets propagated to all child workflows
    # and can have an arbitrary property name.

    parent_wf.run(
        ParentInput(n=2),
        additional_metadata={stream_key: stream_value},
    )


if __name__ == "__main__":
    asyncio.run(main())
