from hatchet_sdk import new_client
from dotenv import load_dotenv
import json
import asyncio

async def main():
    load_dotenv()

    hatchet = new_client()

    workflowRunId = hatchet.admin.run_workflow("ManualTriggerWorkflow", {
        "test": "test"
    })

    listener = hatchet.listener.stream(workflowRunId)

    async for event in listener:
        data = json.dumps({
            "type": event.type,
            "payload": event.payload,
            "messageId": workflowRunId
        })
        print("data: " + data + "\n\n")

if __name__ == "__main__":
    asyncio.run(main())

