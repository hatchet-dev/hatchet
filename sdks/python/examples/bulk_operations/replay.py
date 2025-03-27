from hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus
from datetime import datetime, timedelta

hatchet = Hatchet()

workflows = hatchet.workflows.list()

assert workflows.rows

workflow = workflows.rows[0]

workflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])

workflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]

bulk_cancel_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)

bulk_cancel_by_filters = BulkCancelReplayOpts(
    filters=RunFilter(
        since=datetime.today() - timedelta(days=1),
        until=datetime.now(),
        statuses=[V1TaskStatus.RUNNING],
        workflow_ids=[workflow.metadata.id],
        additional_metadata={"key": "value"},
    )
)
