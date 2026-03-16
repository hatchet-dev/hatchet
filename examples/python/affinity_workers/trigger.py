from examples.affinity_workers.worker import affinity_worker_workflow
from hatchet_sdk import TriggerWorkflowOptions, RunWorkflowOptions

affinity_worker_workflow.run(
    options=RunWorkflowOptions(additional_metadata={"hello": "moon"}),
)
