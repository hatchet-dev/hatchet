from examples.affinity_workers.worker import affinity_worker_workflow
from hatchet_sdk import TriggerWorkflowOptions

affinity_worker_workflow.run(
    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),
)
