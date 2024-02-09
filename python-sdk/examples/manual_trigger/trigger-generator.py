from hatchet_sdk import Hatchet
from dotenv import load_dotenv
import json

class StepRunEventType:
    STEP_RUN_EVENT_TYPE_STARTED = 'STEP_RUN_EVENT_TYPE_STARTED'
    STEP_RUN_EVENT_TYPE_COMPLETED = 'STEP_RUN_EVENT_TYPE_COMPLETED'
    STEP_RUN_EVENT_TYPE_FAILED = 'STEP_RUN_EVENT_TYPE_FAILED'
    STEP_RUN_EVENT_TYPE_CANCELLED = 'STEP_RUN_EVENT_TYPE_CANCELLED'
    STEP_RUN_EVENT_TYPE_TIMED_OUT = 'STEP_RUN_EVENT_TYPE_TIMED_OUT'

load_dotenv()

hatchet = Hatchet().client

workflowRunId = hatchet.admin.run_workflow("ManualTriggerWorkflow", {
    "test": "test"
})

listener = hatchet.listener.stream(workflowRunId)

for event in listener:
    # TODO FIXME step run is not exported easily from the hatchet_sdk and event type and event.step is not defined on
    # the event object, so fix this before merging...

    # if event.step == 'step2' and event.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED:
    #     listener.abort()

    if event.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED:
        print('Step completed: ' + json.dumps(event.payload))

print('Workflow finished')