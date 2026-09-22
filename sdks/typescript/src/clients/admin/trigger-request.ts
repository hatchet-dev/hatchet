import { Code, ConnectError } from '@connectrpc/connect';
import { IdempotencyCollisionError } from '@util/errors/idempotency-collision-error';
import { BulkTriggerIdempotencyCollisionError } from '@util/errors/bulk-trigger-idempotency-collision-error';
import {
  BulkTriggerIdempotencyCollisionErrorSchema,
  IdempotencyCollisionErrorSchema,
} from '@hatchet/protoc-es/v1/workflows_pb';
import type {
  DesiredWorkerLabels,
  TriggerWorkflowRequest,
} from '@hatchet/protoc/v1/shared/trigger';
import type { DeepPartial } from '@clients/transport/message-bridge';
import type { DesiredWorkerLabelOpt, TriggerRunOptions } from '@hatchet/core/types';
import { applyNamespace } from '@hatchet/util/apply-namespace';

/**
 * Builds the `TriggerWorkflow` request for one run: the workflow name is namespaced and
 * lowercased, the input and metadata are JSON, and the deprecated `parentStepRunId` maps onto
 * `parentTaskRunExternalId`. The Node client and the core client send exactly this.
 */
export function buildTriggerWorkflowRequest<Q>(
  workflowName: string,
  input: Q,
  options: TriggerRunOptions | undefined,
  namespace: string | undefined
): DeepPartial<TriggerWorkflowRequest> {
  const opts = options ?? {};

  return {
    name: applyNamespace(workflowName, namespace).toLowerCase(),
    input: JSON.stringify(input),
    parentId: opts.parentId,
    parentTaskRunExternalId: opts.parentTaskRunExternalId ?? opts.parentStepRunId,
    childIndex: opts.childIndex,
    childKey: opts.childKey,
    additionalMetadata: opts.additionalMetadata
      ? JSON.stringify(opts.additionalMetadata)
      : undefined,
    desiredWorkerId: opts.desiredWorkerId,
    priority: opts.priority,
    desiredWorkerLabels: opts.desiredWorkerLabels
      ? convertDesiredWorkerLabels(opts.desiredWorkerLabels)
      : {},
  };
}

function convertDesiredWorkerLabels(
  labels: Record<string, DesiredWorkerLabelOpt>
): Record<string, DesiredWorkerLabels> {
  return Object.fromEntries(
    Object.entries(labels).map(([key, label]) => [
      key,
      {
        strValue: typeof label.value === 'string' ? label.value : undefined,
        intValue: typeof label.value === 'number' ? label.value : undefined,
        required: label.required,
        weight: label.weight,
        comparator: label.comparator,
      } satisfies DesiredWorkerLabels,
    ])
  );
}

/**
 * The engine answers a trigger whose idempotency key is already taken with `ALREADY_EXISTS`
 * and attaches the collision details to the status. The Connect client decodes those details
 * from the trailer, so the error carries everything needed to build the SDK's collision errors.
 */
export function isAlreadyExists(e: unknown): e is ConnectError {
  return e instanceof ConnectError && e.code === Code.AlreadyExists;
}

export function extractExistingRunId(e: ConnectError): string {
  const [detail] = e.findDetails(IdempotencyCollisionErrorSchema);
  return detail?.existingRunExternalId ?? '';
}

export function extractBulkTriggerCollision(
  e: ConnectError
): BulkTriggerIdempotencyCollisionError | null {
  const [detail] = e.findDetails(BulkTriggerIdempotencyCollisionErrorSchema);
  if (!detail) return null;
  return new BulkTriggerIdempotencyCollisionError(
    detail.successfulWorkflowRunExternalIds,
    detail.collisions.map((c) => new IdempotencyCollisionError(c.existingRunExternalId))
  );
}
