import json

from dotenv import load_dotenv

from hatchet_sdk import new_client

load_dotenv()

client = new_client()

workflowRunId = client.admin.run_workflow("ManualTriggerWorkflow", {"test": "test"})

client.listener.on(
    workflowRunId,
    lambda event: print("EVENT: " + event.type + " " + json.dumps(event.payload)),
)
