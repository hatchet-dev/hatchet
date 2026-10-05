from datetime import timedelta

from examples.workflow_pause.worker import pausable_workflow

# > Pause a workflow
pausable_workflow.pause(
    queue_ttl=timedelta(hours=24),
    paused_workflow_cron_run_queue_behavior="DROP",
    paused_workflow_scheduled_run_queue_behavior="QUEUE",
)
# !!

# > Unpause a workflow
pausable_workflow.unpause()
# !!
