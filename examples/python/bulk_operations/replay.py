# > Setup

from datetime import UTC, datetime, timedelta

from hatchet_sdk import BulkCancelReplayOptions, Hatchet, RunFilter

hatchet = Hatchet()

workflows = hatchet.workflows.list()

assert workflows

workflow = workflows[0]


# > List runs
workflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])

# > Replay by run ids
workflow_run_ids = [workflow_run.id for workflow_run in workflow_runs]

bulk_replay_by_ids = BulkCancelReplayOptions(ids=workflow_run_ids)

hatchet.runs.bulk_replay(bulk_replay_by_ids)

# > Replay by filters
bulk_replay_by_filters = BulkCancelReplayOptions(
    filters=RunFilter(
        since=datetime.now(tz=UTC) - timedelta(days=1),
        until=datetime.now(tz=UTC),
        statuses=["RUNNING"],
        workflow_ids=[workflow.metadata.id],
        additional_metadata={"key": "value"},
    )
)

hatchet.runs.bulk_replay(bulk_replay_by_filters)
