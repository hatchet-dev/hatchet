from examples.affinity_workers.worker import affinity_worker_workflow
from examples.bug_tests.payload_bug_on_replay.worker import (
    payload_initial_cancel_bug_workflow,
)
from examples.bulk_fanout.worker import bulk_child_wf, bulk_parent_wf
from examples.bulk_operations.worker import (
    bulk_replay_test_1,
    bulk_replay_test_2,
    bulk_replay_test_3,
)
from examples.cancellation.worker import cancellation_workflow
from examples.concurrency_cancel_in_progress.worker import (
    concurrency_cancel_in_progress_workflow,
)
from examples.concurrency_cancel_newest.worker import concurrency_cancel_newest_workflow
from examples.concurrency_limit.worker import concurrency_limit_workflow
from examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow
from examples.concurrency_multiple_keys.worker import concurrency_multiple_keys_workflow
from examples.concurrency_workflow_level.worker import (
    concurrency_workflow_level_workflow,
)
from examples.conditions.worker import task_condition_workflow
from examples.dag.worker import dag_workflow
from examples.dataclasses.worker import say_hello
from examples.dedupe.worker import dedupe_child_wf, dedupe_parent_wf
from examples.dependency_injection.worker import (
    async_task_with_dependencies,
    di_workflow,
    durable_async_task_with_dependencies,
    sync_task_with_dependencies,
    task_with_type_aliases,
)
from examples.dict_input.worker import say_hello_unsafely
from examples.durable.worker import (
    durable_sleep_event_spawn,
    durable_with_spawn,
    durable_workflow,
    spawn_child_task,
    wait_for_sleep_twice,
    dag_child_workflow,
    durable_spawn_dag,
    durable_non_determinism,
    durable_replay_reset,
    memo_task,
)
from examples.durable_complex.concurrency.worker import (
    durable_concurrency_cancel_in_progress_workflow,
    durable_concurrency_cancel_newest_workflow,
    durable_concurrency_slot_retention_workflow,
    durable_concurrency_workflow,
)
from examples.durable_complex.dag.worker import (
    durable_dag_diamond_workflow,
    durable_dag_durable_parent_workflow,
    durable_dag_parent_failure_workflow,
    durable_dag_workflow,
)
from examples.durable_complex.execution_timeout.worker import (
    durable_refresh_timeout_workflow,
    durable_timeout_completes_workflow,
    durable_timeout_eviction_workflow,
    durable_timeout_workflow,
)
from examples.durable_complex.on_failure.worker import (
    durable_on_failure_details_workflow,
    durable_on_failure_workflow,
    durable_on_success_workflow,
)
from examples.durable_complex.rate_limit.worker import (
    durable_rate_limit_dynamic_workflow,
    durable_rate_limit_workflow,
)
from examples.durable_complex.retries.worker import (
    durable_retries_backoff_workflow,
    durable_retries_exhausted_workflow,
    durable_retries_sleep_workflow,
    durable_retries_workflow,
)
from examples.durable_eviction.worker import (
    child_task as eviction_child_task,
    evictable_child_spawn,
    evictable_sleep,
    evictable_wait_for_event,
    multiple_eviction,
    non_evictable_sleep,
)
from examples.events.worker import event_workflow
from examples.fanout.worker import child_wf, parent_wf
from examples.fanout_sync.worker import sync_fanout_child, sync_fanout_parent
from examples.lifespans.simple import lifespan, lifespan_task
from examples.logger.workflow import logging_workflow
from examples.non_retryable.worker import non_retryable_workflow
from examples.on_failure.worker import on_failure_wf, on_failure_wf_with_details
from examples.return_exceptions.worker import (
    exception_parsing_workflow,
    return_exceptions_task,
)
from examples.run_details.worker import run_detail_test_workflow
from examples.serde.worker import serde_workflow
from examples.simple.worker import simple, simple_durable
from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from examples.webhook_with_scope.worker import (
    webhook_with_scope,
    webhook_with_static_payload,
)
from examples.webhooks.worker import webhook
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
            event_workflow,
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
            concurrency_cancel_newest_workflow,
            concurrency_cancel_in_progress_workflow,
            di_workflow,
            payload_initial_cancel_bug_workflow,
            run_detail_test_workflow,
            lifespan_task,
            simple,
            simple_durable,
            bulk_replay_test_1,
            bulk_replay_test_2,
            bulk_replay_test_3,
            webhook,
            webhook_with_scope,
            webhook_with_static_payload,
            return_exceptions_task,
            exception_parsing_workflow,
            wait_for_sleep_twice,
            spawn_child_task,
            durable_with_spawn,
            durable_sleep_event_spawn,
            async_task_with_dependencies,
            sync_task_with_dependencies,
            durable_async_task_with_dependencies,
            task_with_type_aliases,
            say_hello,
            say_hello_unsafely,
            serde_workflow,
            durable_spawn_dag,
            dag_child_workflow,
            durable_non_determinism,
            durable_replay_reset,
            memo_task,
            evictable_sleep,
            evictable_wait_for_event,
            evictable_child_spawn,
            multiple_eviction,
            non_evictable_sleep,
            eviction_child_task,
            durable_concurrency_workflow,
            durable_concurrency_cancel_in_progress_workflow,
            durable_concurrency_cancel_newest_workflow,
            durable_concurrency_slot_retention_workflow,
            durable_dag_workflow,
            durable_dag_durable_parent_workflow,
            durable_dag_diamond_workflow,
            durable_dag_parent_failure_workflow,
            durable_timeout_workflow,
            durable_timeout_completes_workflow,
            durable_refresh_timeout_workflow,
            durable_timeout_eviction_workflow,
            durable_on_failure_workflow,
            durable_on_success_workflow,
            durable_on_failure_details_workflow,
            durable_rate_limit_workflow,
            durable_rate_limit_dynamic_workflow,
            durable_retries_workflow,
            durable_retries_exhausted_workflow,
            durable_retries_backoff_workflow,
            durable_retries_sleep_workflow,
        ],
        lifespan=lifespan,
    )

    worker.start()


if __name__ == "__main__":
    main()
