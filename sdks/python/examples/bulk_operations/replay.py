# > Setup

from datetime import datetime, timedelta

from hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus

hatchet = Hatchet()

workflows = hatchet.workflows.list()

assert workflows.rows

workflow = workflows.rows[0]

# !!

# > List runs
workflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])
# !!

# > Replay by run ids
workflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]

bulk_replay_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)

hatchet.runs.bulk_replay(bulk_replay_by_ids)
# !!

# > Replay by filters
bulk_replay_by_filters = BulkCancelReplayOpts(
    filters=RunFilter(
        since=datetime.today() - timedelta(days=1),
        until=datetime.now(),
        statuses=[V1TaskStatus.RUNNING],
        workflow_ids=[workflow.metadata.id],
        additional_metadata={"key": "value"},
    )
)

hatchet.runs.bulk_replay(bulk_replay_by_filters)
# !!
