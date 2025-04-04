from examples.priority.worker import priority_workflow
from hatchet_sdk import TriggerWorkflowOptions

priority_workflow.run_no_wait()

low_prio = priority_workflow.run_no_wait(options=TriggerWorkflowOptions(priority=1, additional_metadata={"priority": "low", "key": 1}))
low_prio = priority_workflow.run_no_wait(options=TriggerWorkflowOptions(priority=1, additional_metadata={"priority": "low", "key": 2}))
high_prio = priority_workflow.run_no_wait(options=TriggerWorkflowOptions(priority=3, additional_metadata={"priority": "high", "key": 1}))
high_prio = priority_workflow.run_no_wait(options=TriggerWorkflowOptions(priority=3, additional_metadata={"priority": "high", "key": 2}))
