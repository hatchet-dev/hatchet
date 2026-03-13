# > Setup

from datetime import datetime, timedelta, timezone

from hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus

hatchet = Hatchet()

workflows = await hatchet.workflows.aio_list()

assert workflows.rows

workflow = workflows.rows[0]


# > List runs
workflow_runs = await hatchet.runs.aio_list(workflow_ids=[workflow.metadata.id])

# > Cancel by run ids
workflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]

bulk_cancel_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)

await hatchet.runs.aio_bulk_cancel(bulk_cancel_by_ids)

# > Cancel by filters

bulk_cancel_by_filters = BulkCancelReplayOpts(
    filters=RunFilter(
        since=datetime.today() - timedelta(days=1),
        until=datetime.now(tz=timezone.utc),
        statuses=[V1TaskStatus.RUNNING],
        workflow_ids=[workflow.metadata.id],
        additional_metadata={"key": "value"},
    )
)

await hatchet.runs.aio_bulk_cancel(bulk_cancel_by_filters)
