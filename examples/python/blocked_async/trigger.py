from examples.blocked_async.worker import blocked_worker_workflow
from hatchet_sdk import RunWorkflowOptions

blocked_worker_workflow.run(
    options=RunWorkflowOptions(additional_metadata={"hello": "moon"}),
)
