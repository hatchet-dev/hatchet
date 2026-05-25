/**
 * Shared e2e worker - registers all v1 example workflows for e2e tests.
 * Spawned by jest.e2e-global-setup and killed by jest.e2e-global-teardown.
 * Set HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED=true and
 * HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT before running.
 */
import { hatchet } from './hatchet-client';
import { bulkChild, bulkParentWorkflow } from './bulk_fanout/workflow';
import { bulkReplayTest1, bulkReplayTest2, bulkReplayTest3 } from './bulk_operations/workflow';
import { cancellationWorkflow } from './cancellation/cancellation-workflow';
import { taskConditionWorkflow } from './conditions/complex-workflow';
import { concurrencyCancelInProgressWorkflow } from './concurrency_cancel_in_progress/workflow';
import { concurrencyCancelNewestWorkflow } from './concurrency_cancel_newest/workflow';
import { concurrencyMultipleKeysWorkflow } from './concurrency_multiple_keys/workflow';
import { concurrencyWorkflowLevelWorkflow } from './concurrency_workflow_level/workflow';
import { dag } from './dag/workflow';
import {
  durableWorkflow,
  waitForSleepTwice,
  spawnChildTask,
  durableWithSpawn,
  durableWithBulkSpawn,
  durableSleepEventSpawn,
  durableWithExplicitSpawn,
  durableNonDeterminism,
  durableReplayReset,
  dagChildWorkflow,
  durableSpawnDag,
  waitForEventLookback,
  waitForOrEventLookback,
  waitForTwoEventsSecondPushedFirst,
} from './durable/workflow';
import { durableEvent, durableEventWithFilter } from './durable_event/workflow';
import {
  evictableSleep,
  evictableWaitForEvent,
  evictableChildSpawn,
  multipleEviction,
  nonEvictableSleep,
  childTask as evictionChildTask,
  bulkChildTask,
  evictableChildBulkSpawn,
} from './durable_eviction/workflow';
import { durableSleep } from './durable_sleep/workflow';
import { createLoggingWorkflow } from './logger/workflow';
import { nonRetryableWorkflow } from './non_retryable/workflow';
import { failureWorkflow } from './on_failure/workflow';
import { lower } from './on_event/workflow';
import { returnExceptionsTask } from './return_exceptions/workflow';
import { runDetailTestWorkflow } from './run_details/workflow';
import { helloWorld, helloWorldDurable } from './simple/e2e-workflows';
import { streamingTask } from './streaming/workflow';
import { timeoutTask, refreshTimeoutTask } from './timeout/workflow';
import { webhookWorkflow } from './webhooks/workflow';
import {
  childIndexChild,
  childIndexParent,
  scenarioTask,
  orchestratorTask,
} from './child_index/workflow';
import {
  supportAgent,
  triageTicket,
  generateReply,
  escalateTicket,
} from './support_agent/workflow';
import { welcomeEmail } from './welcome_email/workflow';
import { pdfPipeline } from './pdf_pipeline/workflow';

const workflows = [
  bulkChild,
  bulkParentWorkflow,
  bulkReplayTest1,
  bulkReplayTest2,
  bulkReplayTest3,
  cancellationWorkflow,
  taskConditionWorkflow,
  concurrencyCancelInProgressWorkflow,
  concurrencyCancelNewestWorkflow,
  concurrencyMultipleKeysWorkflow,
  concurrencyWorkflowLevelWorkflow,
  dag,
  durableWorkflow,
  waitForSleepTwice,
  spawnChildTask,
  durableWithSpawn,
  durableWithBulkSpawn,
  durableSleepEventSpawn,
  durableWithExplicitSpawn,
  durableNonDeterminism,
  durableReplayReset,
  dagChildWorkflow,
  durableSpawnDag,
  waitForEventLookback,
  waitForOrEventLookback,
  waitForTwoEventsSecondPushedFirst,
  durableEvent,
  durableEventWithFilter,
  durableSleep,
  evictableSleep,
  evictableWaitForEvent,
  evictableChildSpawn,
  multipleEviction,
  nonEvictableSleep,
  evictionChildTask,
  bulkChildTask,
  evictableChildBulkSpawn,
  createLoggingWorkflow(hatchet),
  nonRetryableWorkflow,
  failureWorkflow,
  lower,
  returnExceptionsTask,
  runDetailTestWorkflow,
  helloWorld,
  helloWorldDurable,
  streamingTask,
  timeoutTask,
  refreshTimeoutTask,
  webhookWorkflow,
  childIndexChild,
  childIndexParent,
  scenarioTask,
  orchestratorTask,
  supportAgent,
  triageTicket,
  generateReply,
  escalateTicket,
  welcomeEmail,
  pdfPipeline,
];

async function main() {
  const worker = await hatchet.worker('e2e-test-worker', {
    workflows,
    slots: 100,
  });

  void worker.start();
  await worker.waitUntilReady(30_000);
  console.log('[e2e-worker] Worker registered and ready');

  await new Promise(() => {});
}

main().catch((err) => {
  console.error('e2e-worker failed:', err);
  process.exit(1);
});
