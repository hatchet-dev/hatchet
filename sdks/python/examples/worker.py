from examples.affinity_workers.worker import affinity_worker_workflow
from examples.bulk_fanout.worker import bulk_child_wf, bulk_parent_wf
from examples.cancellation.worker import cancellation_workflow
from examples.concurrency_limit.worker import concurrency_limit_workflow
from examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow
from examples.concurrency_multiple_keys.worker import concurrency_multiple_keys_workflow
from examples.concurrency_workflow_level.worker import (
    concurrency_workflow_level_workflow,
)
from examples.dag.worker import dag_workflow
from examples.dedupe.worker import dedupe_child_wf, dedupe_parent_wf
from examples.durable.worker import durable_workflow
from examples.fanout.worker import child_wf, parent_wf
from examples.fanout_sync.worker import sync_fanout_child, sync_fanout_parent
from examples.lifespans.simple import lifespan, lifespan_task
from examples.logger.workflow import logging_workflow
from examples.non_retryable.worker import non_retryable_workflow
from examples.on_failure.worker import on_failure_wf, on_failure_wf_with_details
from examples.priority.worker import priority_workflow
from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from examples.waits.worker import task_condition_workflow
from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


def main() -> None:
    worker = hatchet.worker(
        "e2e-test-worker",
        slots=100,
        workflows=[
            affinity_worker_workflow,
            bulk_child_wf,
            bulk_parent_wf,
            concurrency_limit_workflow,
            concurrency_limit_rr_workflow,
            concurrency_multiple_keys_workflow,
            dag_workflow,
            dedupe_child_wf,
            dedupe_parent_wf,
            durable_workflow,
            child_wf,
            parent_wf,
            on_failure_wf,
            on_failure_wf_with_details,
            logging_workflow,
            timeout_wf,
            refresh_timeout_wf,
            task_condition_workflow,
            cancellation_workflow,
            sync_fanout_parent,
            sync_fanout_child,
            non_retryable_workflow,
            concurrency_workflow_level_workflow,
            priority_workflow,
            lifespan_task,
        ],
        lifespan=lifespan,
    )

    worker.start()


if __name__ == "__main__":
    main()
