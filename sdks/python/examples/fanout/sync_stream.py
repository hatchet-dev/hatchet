import random

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions


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

    hatchet.admin.run_workflow(
        "Parent",
        {"n": 2},
        options=TriggerWorkflowOptions(additional_metadata={streamKey: streamVal}),
    )

    # Stream all events for the additional meta key value
    listener = hatchet.listener.stream_by_additional_metadata(streamKey, streamVal)

    for event in listener:
        print(event.type, event.payload)

    print("DONE.")


if __name__ == "__main__":
    main()
