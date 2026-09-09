import random

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet


def main() -> None:

    hatchet = Hatchet()

    # Generate a random stream key to use to track all
    # stream events for this workflow run.

    streamKey = "streamKey"
    streamVal = f"sk-{random.randint(1, 100)}"

    # Specify the stream key as additional metadata
    # when running the workflow.

    # This key gets propagated to all child workflows
    # and can have an arbitrary property name.

    parent_wf.run(
        ParentInput(n=2),
        additional_metadata={streamKey: streamVal},
    )

    print("DONE.")


if __name__ == "__main__":
    main()
