import HatchetError from '@util/errors/hatchet-error';
import { IdempotencyCollisionError } from '@util/errors/idempotency-collision-error';
import { BulkTriggerIdempotencyCollisionError } from '@util/errors/bulk-trigger-idempotency-collision-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Code, ConnectError } from '@connectrpc/connect';
import {
  BulkTriggerIdempotencyCollisionErrorSchema,
  IdempotencyCollisionErrorSchema,
} from '@hatchet/protoc-es/v1/workflows_pb';
import { createClientFactory } from 'nice-grpc';

import { Priority, RateLimitDuration, RunsClient, WorkerLabelComparator } from '@hatchet/v1';
import { addTokenMiddleware, channelFactory } from '@hatchet/util/grpc-helpers';
import { ConfigLoader } from '@hatchet/util/config-loader';
import { RunListenerClient } from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { Api } from '@hatchet/clients/rest/generated/Api';
import { BulkTriggerWorkflowRequest, WorkflowServiceClient } from '@hatchet/protoc/workflows';
import { AdminServiceClient, CreateWorkflowVersionRequest } from '@hatchet/protoc/v1/workflows';
import { createV1AdminRpc, createWorkflowsRpc } from '@hatchet/clients/admin/rpc';
import { createNodeTransport, type Transport } from '@hatchet/clients/transport';
import { Logger } from '@hatchet/util/logger';
import { retrier } from '@hatchet/util/retrier';
import { batch } from '@hatchet/util/batch';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { DesiredWorkerLabels } from '@hatchet-dev/typescript-sdk/protoc/v1/shared/trigger';

/**
 * The engine answers a trigger whose idempotency key is already taken with `ALREADY_EXISTS`
 * and attaches the collision details to the status. The Connect client decodes those details
 * from the trailer, so the error carries everything needed to build the SDK's collision errors.
 */
function isAlreadyExists(e: unknown): e is ConnectError {
  return e instanceof ConnectError && e.code === Code.AlreadyExists;
}

function extractExistingRunId(e: ConnectError): string {
  const [detail] = e.findDetails(IdempotencyCollisionErrorSchema);
  return detail?.existingRunExternalId ?? '';
}

function extractBulkTriggerCollision(e: ConnectError): BulkTriggerIdempotencyCollisionError | null {
  const [detail] = e.findDetails(BulkTriggerIdempotencyCollisionErrorSchema);
  if (!detail) return null;
  return new BulkTriggerIdempotencyCollisionError(
    detail.successfulWorkflowRunExternalIds,
    detail.collisions.map((c) => new IdempotencyCollisionError(c.existingRunExternalId))
  );
}

type DesiredWorkerLabelOpt = {
  value: string | number;
  required?: boolean;
  weight?: number;
  comparator?: WorkerLabelComparator;
};

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

export type WorkflowRun<T = object> = {
  workflowName: string;
  input: T;
  options?: {
    parentId?: string | undefined;
    /**
     * (optional) the parent task run external id.
     *
     * This is the field understood by the workflows gRPC API (`parent_task_run_external_id`).
     */
    parentTaskRunExternalId?: string | undefined;
    /**
     * @deprecated Use `parentTaskRunExternalId` instead.
     * Kept for backward compatibility; will be mapped to `parentTaskRunExternalId`.
     */
    parentStepRunId?: string | undefined;
    childIndex?: number | undefined;
    childKey?: string | undefined;
    additionalMetadata?: Record<string, string> | undefined;
  };
};

export class AdminClient {
  config: ClientConfig;
  workflowsGrpc: WorkflowServiceClient;
  adminGrpc: AdminServiceClient;
  listenerClient: RunListenerClient;
  runs: RunsClient;
  logger: Logger;

  constructor(
    config: ClientConfig,
    api: Api,
    runs: RunsClient,
    transport: Transport = createNodeTransport(config)
  ) {
    this.config = config;
    this.logger = config.logger(`Admin`, config.log_level);

    this.workflowsGrpc = createWorkflowsRpc(transport);
    this.adminGrpc = createV1AdminRpc(transport);

    // The run listener streams, which stay on the nice-grpc channel.
    const channel = channelFactory(config, ConfigLoader.createCredentials(config.tls_config));
    const factory = createClientFactory().use(addTokenMiddleware(config.token));
    this.listenerClient = new RunListenerClient(config, channel, factory, api);
    this.runs = runs;
  }

  /**
   * Creates a new workflow or updates an existing workflow via the v1 admin service.
   * @param workflow a workflow definition to create
   */
  async putWorkflow(workflow: CreateWorkflowVersionRequest) {
    try {
      return await retrier(async () => this.adminGrpc.putWorkflow(workflow), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Run a new instance of a workflow with the given input. This will create a new workflow run and return the ID of the
   * new run.
   * @param workflowName the name of the workflow to run
   * @param input an object containing the input to the workflow
   * @param options an object containing the options to run the workflow
   * @returns the ID of the new workflow run
   *
   * @important This method is instrumented by HatchetInstrumentor._patchRunWorkflow.
   * Keep the signature in sync with the instrumentor wrapper.
   */
  async runWorkflow<Q = object, P = object>(
    workflowName: string,
    input: Q,
    options?: {
      parentId?: string | undefined;
      /**
       * (optional) the parent task run external id.
       *
       * This is the field understood by the workflows gRPC API (`parent_task_run_external_id`).
       */
      parentTaskRunExternalId?: string | undefined;
      /**
       * @deprecated Use `parentTaskRunExternalId` instead.
       * Kept for backward compatibility; will be mapped to `parentTaskRunExternalId`.
       */
      parentStepRunId?: string | undefined;
      childIndex?: number | undefined;
      childKey?: string | undefined;
      additionalMetadata?: Record<string, string> | undefined;
      desiredWorkerId?: string | undefined;
      priority?: Priority;
      desiredWorkerLabels?: Record<string, DesiredWorkerLabelOpt>;
      _standaloneTaskName?: string | undefined;
    }
  ) {
    try {
      const computedName = applyNamespace(workflowName, this.config.namespace).toLowerCase();

      const inputStr = JSON.stringify(input);

      const opts = options ?? {};
      const {
        additionalMetadata,
        parentStepRunId,
        parentTaskRunExternalId,
        desiredWorkerLabels,
        ...rest
      } = opts;

      const request = {
        name: computedName,
        input: inputStr,
        ...rest,
        // API expects `parentTaskRunExternalId`; accept old names as aliases.
        parentTaskRunExternalId: parentTaskRunExternalId ?? parentStepRunId,
        additionalMetadata: additionalMetadata ? JSON.stringify(additionalMetadata) : undefined,
        priority: opts.priority,
        desiredWorkerLabels: desiredWorkerLabels
          ? convertDesiredWorkerLabels(desiredWorkerLabels)
          : {},
      };

      const resp = await retrier(
        async () => this.workflowsGrpc.triggerWorkflow(request),
        this.logger,
        { ...this.config.retrier, shouldRetry: (e) => !isAlreadyExists(e) }
      );

      const id = resp.workflowRunId;

      const ref = new WorkflowRunRef<P>(
        id,
        this.listenerClient,
        this.runs,
        options?.parentId,

        options?._standaloneTaskName
      );
      await ref.getWorkflowRunId();
      return ref;
    } catch (e: unknown) {
      if (isAlreadyExists(e)) {
        throw new IdempotencyCollisionError(extractExistingRunId(e));
      }
      throw new HatchetError(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Run multiple workflows runs with the given input and options. This will create new workflow runs and return their IDs.
   * Order is preserved in the response.
   * @param workflowRuns an array of objects containing the workflow name, input, and options for each workflow run
   * @returns an array of workflow run references
   *
   * @important This method is instrumented by HatchetInstrumentor._patchRunWorkflows.
   * Keep the signature in sync with the instrumentor wrapper.
   */
  async runWorkflows<Q = object, P = object>(
    workflowRuns: Array<{
      workflowName: string;
      input: Q;
      options?: {
        parentId?: string | undefined;
        /**
         * (optional) the parent task run external id.
         *
         * This is the field understood by the workflows gRPC API (`parent_task_run_external_id`).
         */
        parentTaskRunExternalId?: string | undefined;
        /**
         * @deprecated Use `parentTaskRunExternalId` instead.
         * Kept for backward compatibility; will be mapped to `parentTaskRunExternalId`.
         */
        parentStepRunId?: string | undefined;
        childIndex?: number | undefined;
        childKey?: string | undefined;
        additionalMetadata?: Record<string, string> | undefined;
        desiredWorkerId?: string | undefined;
        priority?: Priority;
        desiredWorkerLabels?: Record<string, DesiredWorkerLabelOpt>;
        _standaloneTaskName?: string | undefined;
      };
    }>,
    batchSize: number = 500
  ): Promise<WorkflowRunRef<P>[]> {
    // Prepare workflows to be triggered in bulk
    const workflowRequests = workflowRuns.map(({ workflowName, input, options }) => {
      const computedName = applyNamespace(workflowName, this.config.namespace).toLowerCase();
      const inputStr = JSON.stringify(input);

      const opts = options ?? {};
      const {
        additionalMetadata,
        parentStepRunId,
        parentTaskRunExternalId,
        desiredWorkerLabels,
        ...rest
      } = opts;

      return {
        name: computedName,
        input: inputStr,
        ...rest,
        // API expects `parentTaskRunExternalId`; accept old names as aliases.
        parentTaskRunExternalId: parentTaskRunExternalId ?? parentStepRunId,
        additionalMetadata: additionalMetadata ? JSON.stringify(additionalMetadata) : undefined,
        desiredWorkerLabels: desiredWorkerLabels
          ? convertDesiredWorkerLabels(desiredWorkerLabels)
          : {},
      };
    });

    const batches = batch(
      workflowRequests,
      batchSize,
      this.config.grpc_max_send_message_length ?? 4 * 1024 * 1024
    );

    this.logger.debug(`batching ${batches.length} batches`);

    try {
      const results: WorkflowRunRef<P>[] = [];

      // for loop to ensure serial execution of batches
      for (const { payloads, originalIndices, batchIndex } of batches) {
        const request = BulkTriggerWorkflowRequest.create({
          workflows: payloads,
        });

        // Call the bulk trigger workflow method for this batch
        const bulkTriggerWorkflowResponse = await retrier(
          async () => this.workflowsGrpc.bulkTriggerWorkflow(request),
          this.logger,
          { ...this.config.retrier, shouldRetry: (e) => !isAlreadyExists(e) }
        );

        this.logger.debug(`batch ${batchIndex + 1} of ${batches.length}`);

        // Map the results back to their original indices
        const batchResults = bulkTriggerWorkflowResponse.workflowRunIds.map((resp, index) => {
          const originalIndex = originalIndices[index];
          const { options } = workflowRuns[originalIndex];
          return new WorkflowRunRef<P>(
            resp,
            this.listenerClient,
            this.runs,
            options?.parentId,

            options?._standaloneTaskName
          );
        });

        results.push(...batchResults);
      }
      return results;
    } catch (e: unknown) {
      if (isAlreadyExists(e)) {
        const collision = extractBulkTriggerCollision(e);
        if (collision) throw collision;
      }
      throw new HatchetError(e instanceof Error ? (e as Error).message : String(e));
    }
  }

  async putRateLimit(key: string, limit: number, duration?: RateLimitDuration) {
    const request = {
      key,
      limit,
      duration,
    };

    await retrier(
      async () => this.workflowsGrpc.putRateLimit(request),
      this.logger,
      this.config.retrier
    );
  }
}
