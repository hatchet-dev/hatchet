from examples.sticky_workers.worker import sticky_workflow
from hatchet_sdk import TriggerWorkflowOptions

sticky_workflow.run(
    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
)
