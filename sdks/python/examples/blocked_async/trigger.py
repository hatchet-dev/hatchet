from examples.blocked_async.worker import blocked_worker_workflow
from hatchet_sdk import TriggerWorkflowOptions

blocked_worker_workflow.run(
    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
)
