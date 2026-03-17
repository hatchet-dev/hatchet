from datetime import datetime, timedelta, timezone

from examples.priority.worker import priority_workflow
from hatchet_sdk import (
    TriggerWorkflowOptions,
    Priority,
    ScheduleTriggerWorkflowOptions,
    ScheduleTriggerWorkflowOptions,
)

priority_workflow.run(wait_for_result=False)

# > Runtime priority
low_prio = priority_workflow.run(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=Priority.LOW,
        additional_metadata={"priority": "low", "key": 1},
    ),
    wait_for_result=False,
)

high_prio = priority_workflow.run(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=Priority.HIGH,
        additional_metadata={"priority": "high", "key": 1},
    ),
    wait_for_result=False,
)
# !!

# > Scheduled priority
schedule = priority_workflow.schedule(
    run_at=datetime.now(tz=timezone.utc) + timedelta(minutes=1),
    options=ScheduleTriggerWorkflowOptions(priority=Priority.HIGH),
)

cron = priority_workflow.create_cron(
    cron_name="my-scheduled-cron",
    expression="0 * * * *",
    priority=Priority.HIGH,
)
# !!

# > Default priority
low_prio = priority_workflow.run(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=Priority.LOW,
        additional_metadata={"priority": "low", "key": 2},
    ),
    wait_for_result=False,
)
high_prio = priority_workflow.run(
    options=TriggerWorkflowOptions(
        ## 👀 Adding priority and key to metadata to show them in the dashboard
        priority=Priority.HIGH,
        additional_metadata={"priority": "high", "key": 2},
    ),
    wait_for_result=False,
)
