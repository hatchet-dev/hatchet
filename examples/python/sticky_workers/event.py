from examples.sticky_workers.worker import sticky_workflow
from hatchet_sdk import RunWorkflowOptions

sticky_workflow.run(
    options=RunWorkflowOptions(additional_metadata={"hello": "moon"}),
)
