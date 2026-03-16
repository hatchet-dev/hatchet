from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import RunWorkflowOptions

bulk_parent_wf.run(
    ParentInput(n=999),
    RunWorkflowOptions(additional_metadata={"no-dedupe": "world"}),
)
