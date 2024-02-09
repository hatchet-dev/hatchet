from hatchet_sdk import Hatchet
from dotenv import load_dotenv
import json

# StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED

load_dotenv()

hatchet = Hatchet().client

workflowRunId = hatchet.admin.run_workflow("ManualTriggerWorkflow", {
    "test": "test"
})

listener = hatchet.listener.stream(workflowRunId)

for event in listener:
    # TODO FIXME step run is not exported easily from the hatchet_sdk and event type and event.step is not defined on
    # the event object, so fix this before merging...

    if event.step == 'step2' and event.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED:
        listener.abort()
    print('EVENT: ' + event.type + ' ' + json.dumps(event.payload))
