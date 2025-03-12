from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import TriggerWorkflowOptions

bulk_parent_wf.run(
    ParentInput(n=999),
    TriggerWorkflowOptions(additional_metadata={"no-dedupe": "world"}),
)
