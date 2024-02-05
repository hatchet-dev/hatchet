from hatchet_sdk import Hatchet
from dotenv import load_dotenv
import json

load_dotenv()

client = Hatchet().client

workflowRunId = client.admin.run_workflow("ManualTriggerWorkflow", {
    "test": "test"
})

for event in client.listener.generator(workflowRunId):
    print('EVENT: ' + event.type + ' ' + json.dumps(event.payload))

# TODO - need to hangup the listener if the workflow is completed
