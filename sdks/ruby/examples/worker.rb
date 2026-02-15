# frozen_string_literal: true

# Main worker that registers all example workflows.

require "hatchet-sdk"

# Load all example workflows
require_relative "simple/worker"
require_relative "dag/worker"
require_relative "events/worker"
require_relative "cancellation/worker"
require_relative "on_failure/worker"
require_relative "on_success/worker"
require_relative "timeout/worker"
require_relative "retries/worker"
require_relative "non_retryable/worker"
require_relative "logger/worker"
require_relative "delayed/worker"
require_relative "priority/worker"
require_relative "run_details/worker"
require_relative "concurrency_limit/worker"
require_relative "concurrency_limit_rr/worker"
require_relative "concurrency_cancel_in_progress/worker"
require_relative "concurrency_cancel_newest/worker"
require_relative "concurrency_multiple_keys/worker"
require_relative "concurrency_workflow_level/worker"
require_relative "rate_limit/worker"
require_relative "child/worker"
require_relative "fanout/worker"
require_relative "bulk_fanout/worker"
require_relative "durable/worker"
require_relative "durable_event/worker"
require_relative "durable_sleep/worker"
require_relative "conditions/worker"
require_relative "dependency_injection/worker"
require_relative "lifespans/worker"
require_relative "streaming/worker"
require_relative "serde/worker"
require_relative "dataclasses/worker"
require_relative "dedupe/worker"
require_relative "cron/worker"
require_relative "scheduled/worker"
require_relative "bulk_operations/worker"
require_relative "return_exceptions/worker"
require_relative "manual_slot_release/worker"
require_relative "affinity_workers/worker"
require_relative "sticky_workers/worker"
require_relative "webhooks/worker"
require_relative "webhook_with_scope/worker"
require_relative "unit_testing/worker"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

ALL_WORKFLOWS = [
  # Tier 1
  SIMPLE, SIMPLE_DURABLE,
  DAG_WORKFLOW,
  EVENT_WORKFLOW,
  CANCELLATION_WORKFLOW,
  ON_FAILURE_WF, ON_FAILURE_WF_WITH_DETAILS,
  ON_SUCCESS_WORKFLOW,
  TIMEOUT_WF, REFRESH_TIMEOUT_WF,
  SIMPLE_RETRY_WORKFLOW, BACKOFF_WORKFLOW,
  NON_RETRYABLE_WORKFLOW,
  LOGGING_WORKFLOW,
  PRINT_SCHEDULE_WF, PRINT_PRINTER_WF,
  PRIORITY_WORKFLOW,
  RUN_DETAIL_TEST_WORKFLOW,

  # Tier 2
  CONCURRENCY_LIMIT_WORKFLOW,
  CONCURRENCY_LIMIT_RR_WORKFLOW,
  CONCURRENCY_CANCEL_IN_PROGRESS_WORKFLOW,
  CONCURRENCY_CANCEL_NEWEST_WORKFLOW,
  CONCURRENCY_MULTIPLE_KEYS_WORKFLOW,
  CONCURRENCY_WORKFLOW_LEVEL_WORKFLOW,
  RATE_LIMIT_WORKFLOW,

  # Tier 3
  CHILD_TASK_WF,
  FANOUT_PARENT_WF, FANOUT_CHILD_WF,
  BULK_PARENT_WF, BULK_CHILD_WF,
  DURABLE_WORKFLOW, EPHEMERAL_WORKFLOW, WAIT_FOR_SLEEP_TWICE,
  DURABLE_EVENT_TASK, DURABLE_EVENT_TASK_WITH_FILTER,
  DURABLE_SLEEP_TASK,
  TASK_CONDITION_WORKFLOW,
  ASYNC_TASK_WITH_DEPS, SYNC_TASK_WITH_DEPS,
  DURABLE_ASYNC_TASK_WITH_DEPS, DURABLE_SYNC_TASK_WITH_DEPS,
  DI_WORKFLOW,
  LIFESPAN_TASK,

  # Tier 4-5
  STREAM_TASK,
  SERDE_WORKFLOW,
  SAY_HELLO,
  DEDUPE_PARENT_WF, DEDUPE_CHILD_WF,
  CRON_WORKFLOW,
  SCHEDULED_WORKFLOW,
  BULK_REPLAY_TEST_1, BULK_REPLAY_TEST_2, BULK_REPLAY_TEST_3,
  RETURN_EXCEPTIONS_TASK,
  SLOT_RELEASE_WORKFLOW,
  AFFINITY_WORKER_WORKFLOW,
  STICKY_WORKFLOW, STICKY_CHILD_WORKFLOW,
  WEBHOOK_TASK,
  WEBHOOK_WITH_SCOPE, WEBHOOK_WITH_STATIC_PAYLOAD,
  SYNC_STANDALONE, ASYNC_STANDALONE,
  DURABLE_SYNC_STANDALONE, DURABLE_ASYNC_STANDALONE,
  SIMPLE_UNIT_TEST_WORKFLOW, COMPLEX_UNIT_TEST_WORKFLOW
].freeze

worker = HATCHET.worker("all-examples-worker", slots: 40, workflows: ALL_WORKFLOWS)
worker.start
