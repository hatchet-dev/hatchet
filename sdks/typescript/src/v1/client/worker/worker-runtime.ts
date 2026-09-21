/**
 * `ContextRuntime` for a worker process: adapts the `HatchetClient` and `InternalWorker`
 * a task runs on to the seam `Context` consumes.
 */
import { StepActionEventType } from '@hatchet/protoc/dispatcher';
import type { LogLevel as EventLogLevel } from '@hatchet/clients/event/event-client';
import type { HatchetClient } from '@hatchet/v1/client/client';
import type { LogLevel } from '@hatchet/util/logger';
import type { InternalWorker } from './worker-internal';
import type { ContextRuntime, SpawnRunOptions, SpawnRunRequest, WorkerLabels } from './runtime';

/** A `ContextRuntime` that also exposes the client it wraps, for `ctx.v1`. */
export interface WorkerContextRuntime extends ContextRuntime {
  readonly client: HatchetClient;
}

// The event client's LogLevel enum has the same string values as the logger's LogLevel
// union; casting keeps the event client (and its gRPC imports) out of this module's
// runtime graph.
function toEventLogLevel(level: LogLevel | undefined): EventLogLevel | undefined {
  if (level === undefined || level === 'OFF') {
    return undefined;
  }
  return level as unknown as EventLogLevel;
}

export function createWorkerContextRuntime(
  client: HatchetClient,
  worker: InternalWorker
): WorkerContextRuntime {
  return {
    client,

    get namespace() {
      return client.config.namespace;
    },

    logger: (name: string) => client.config.logger(name, client.config.log_level),

    cancelRun: async (taskRunExternalId: string) => {
      await client.runs.cancel({ ids: [taskRunExternalId] });
    },

    cancelBatch: async ({ workerId, jobId, actionId, batchId, memberIds }) => {
      await client.dispatcher.sendBatchActionEvent({
        workerId,
        jobId,
        actionId,
        batchId,
        eventTimestamp: new Date(),
        eventType: StepActionEventType.STEP_EVENT_TYPE_CANCELLED,
        items: memberIds.map((id) => ({ taskRunExternalId: id, eventPayload: '' })),
      });
    },

    putLog: async (taskRunExternalId, message, level, retryCount, extra) => {
      await client.events.putLog(
        taskRunExternalId,
        message,
        toEventLogLevel(level),
        retryCount,
        extra
      );
    },

    refreshTimeout: async (taskRunExternalId, incrementBy) => {
      await client.dispatcher.refreshTimeout(incrementBy, taskRunExternalId);
    },

    releaseSlot: async (taskRunExternalId) => {
      await client.dispatcher.client.releaseSlot({ taskRunExternalId });
    },

    putStream: async (taskRunExternalId, data, index) => {
      await client.events.putStream(taskRunExternalId, data, index);
    },

    runWorkflow: <Q = object, P = object>(
      workflowName: string,
      input: Q,
      options?: SpawnRunOptions
    ) => client.admin.runWorkflow<Q, P>(workflowName, input, options),

    runWorkflows: <Q = object, P = object>(runs: SpawnRunRequest<Q>[]) =>
      client.admin.runWorkflows<Q, P>(runs),

    workerId: () => worker.workerId,

    hasWorkflow: (workflowName: string) =>
      !!worker.workflow_registry.find((workflow) =>
        'id' in workflow ? workflow.id === workflowName : workflow.name === workflowName
      ),

    workerLabels: (): WorkerLabels => worker.labels,

    upsertWorkerLabels: (labels: WorkerLabels) => worker.upsertLabels(labels),
  };
}
