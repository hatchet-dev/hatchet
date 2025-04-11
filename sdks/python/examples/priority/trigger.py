from datetime import datetime, timedelta

from examples.priority.worker import priority_workflow
from hatchet_sdk import ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions

priority_workflow.run_no_wait()

# ❓ Runtime priority
low_prio = priority_workflow.run_no_wait(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=1,
        additional_metadata={"priority": "low", "key": 1},
    )
)

high_prio = priority_workflow.run_no_wait(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=3,
        additional_metadata={"priority": "high", "key": 1},
    )
)
# !!

# ❓ Scheduled priority
schedule = priority_workflow.schedule(
    run_at=datetime.now() + timedelta(minutes=1),
    options=ScheduleTriggerWorkflowOptions(priority=3),
)

cron = priority_workflow.create_cron(
    cron_name="my-scheduled-cron",
    expression="0 * * * *",
    priority=3,
)
# !!

# ❓ Default priority
low_prio = priority_workflow.run_no_wait(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=1,
        additional_metadata={"priority": "low", "key": 2},
    )
)
high_prio = priority_workflow.run_no_wait(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=3,
        additional_metadata={"priority": "high", "key": 2},
    )
)
