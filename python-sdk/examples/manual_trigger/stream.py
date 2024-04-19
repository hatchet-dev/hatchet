import asyncio
import base64
import json
import os

from dotenv import load_dotenv

from hatchet_sdk import new_client
from hatchet_sdk.clients.listener import StepRunEventType


async def main():
    load_dotenv()
    hatchet = new_client()
    workflowRunId = hatchet.admin.run_workflow(
        "ManualTriggerWorkflow", {"test": "test"}
    )
    listener = hatchet.listener.stream(workflowRunId)

    # Get the directory of the current script
    script_dir = os.path.dirname(os.path.abspath(__file__))

    # Create the "out" directory if it doesn't exist
    out_dir = os.path.join(script_dir, "out")
    os.makedirs(out_dir, exist_ok=True)

    async for event in listener:
        if event.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            # Decode the base64-encoded payload
            decoded_payload = base64.b64decode(event.payload)

            # Construct the path to the payload file in the "out" directory
            payload_path = os.path.join(out_dir, "payload.jpg")

            with open(payload_path, "wb") as f:
                f.write(decoded_payload)

            data = json.dumps({"type": event.type, "messageId": workflowRunId})
            print("data: " + data + "\n\n")


if __name__ == "__main__":
    asyncio.run(main())
