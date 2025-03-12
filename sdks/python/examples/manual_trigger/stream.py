import asyncio
import base64
import json
import os

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions
from hatchet_sdk.clients.run_event_listener import StepRunEventType

hatchet = Hatchet()


async def main() -> None:
    workflowRun = hatchet.admin.run_workflow(
        "ManualTriggerWorkflow",
        {"test": "test"},
        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
    )

    listener = workflowRun.stream()

    # Get the directory of the current script
    script_dir = os.path.dirname(os.path.abspath(__file__))

    # Create the "out" directory if it doesn't exist
    out_dir = os.path.join(script_dir, "out")
    os.makedirs(out_dir, exist_ok=True)

    async for event in listener:
        print(event.type, event.payload)
        if event.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            # Decode the base64-encoded payload
            decoded_payload = base64.b64decode(event.payload)

            # Construct the path to the payload file in the "out" directory
            payload_path = os.path.join(out_dir, "payload.jpg")

            with open(payload_path, "wb") as f:
                f.write(decoded_payload)

            data = json.dumps(
                {"type": event.type, "messageId": workflowRun.workflow_run_id}
            )
            print("data: " + data + "\n\n")

    result = await workflowRun.aio_result()

    print("result: " + json.dumps(result, indent=2))


if __name__ == "__main__":
    asyncio.run(main())
