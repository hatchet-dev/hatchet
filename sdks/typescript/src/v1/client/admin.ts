import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';

import { Priority, RateLimitDuration, RunsClient } from '@hatchet/v1';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { RunListenerClient } from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { Api } from '@hatchet/clients/rest/generated/Api';
import {
  BulkTriggerWorkflowRequest,
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@hatchet/protoc/workflows';
import { Logger } from '@hatchet/util/logger';
import { batch } from '@hatchet/util/batch';
import { applyNamespace } from '@hatchet/util/apply-namespace';

export type WorkflowRun<T = object> = {
  workflowName: string;
  input: T;
  options?: {
    parentId?: string | undefined;
    parentStepRunId?: string | undefined;
    childIndex?: number | undefined;
    childKey?: string | undefined;
    additionalMetadata?: Record<string, string> | undefined;
  };
};

export class AdminClient {
  config: ClientConfig;
  grpc: WorkflowServiceClient;
  listenerClient: RunListenerClient;
  runs: RunsClient;
  logger: Logger;

  constructor(config: ClientConfig, api: Api, runs: RunsClient) {
    this.config = config;
    this.logger = config.logger(`Admin`, config.log_level);

    const { client, channel, factory } = createGrpcClient(config, WorkflowServiceDefinition);
    this.grpc = client;
    this.listenerClient = new RunListenerClient(config, channel, factory, api);
    this.runs = runs;
  }

  /**
   * Run a new instance of a workflow with the given input. This will create a new workflow run and return the ID of the
   * new run.
   * @param workflowName the name of the workflow to run
   * @param input an object containing the input to the workflow
   * @param options an object containing the options to run the workflow
   * @returns the ID of the new workflow run
   */
  async runWorkflow<Q = object, P = object>(
    workflowName: string,
    input: Q,
    options?: {
      parentId?: string | undefined;
      parentStepRunId?: string | undefined;
      childIndex?: number | undefined;
      childKey?: string | undefined;
      additionalMetadata?: Record<string, string> | undefined;
      desiredWorkerId?: string | undefined;
      priority?: Priority;
      _standaloneTaskName?: string | undefined;
    }
  ) {
    try {
      const computedName = applyNamespace(workflowName, this.config.namespace);

      const inputStr = JSON.stringify(input);

      const resp = await this.grpc.triggerWorkflow({
        name: computedName,
        input: inputStr,
        ...options,
        additionalMetadata: options?.additionalMetadata
          ? JSON.stringify(options?.additionalMetadata)
          : undefined,
        priority: options?.priority,
      });

      const id = resp.workflowRunId;

      const ref = new WorkflowRunRef<P>(
        id,
        this.listenerClient,
        this.runs,
        options?.parentId,
        // eslint-disable-next-line no-underscore-dangle
        options?._standaloneTaskName
      );
      await ref.getWorkflowRunId();
      return ref;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  /**
   * Run multiple workflows runs with the given input and options. This will create new workflow runs and return their IDs.
   * Order is preserved in the response.
   * @param workflowRuns an array of objects containing the workflow name, input, and options for each workflow run
   * @returns an array of workflow run references
   */
  async runWorkflows<Q = object, P = object>(
    workflowRuns: Array<{
      workflowName: string;
      input: Q;
      options?: {
        parentId?: string | undefined;
        parentStepRunId?: string | undefined;
        childIndex?: number | undefined;
        childKey?: string | undefined;
        additionalMetadata?: Record<string, string> | undefined;
        desiredWorkerId?: string | undefined;
        priority?: Priority;
        _standaloneTaskName?: string | undefined;
      };
    }>,
    batchSize: number = 500
  ): Promise<WorkflowRunRef<P>[]> {
    // Prepare workflows to be triggered in bulk
    const workflowRequests = workflowRuns.map(({ workflowName, input, options }) => {
      const computedName = applyNamespace(workflowName, this.config.namespace);
      const inputStr = JSON.stringify(input);

      return {
        name: computedName,
        input: inputStr,
        ...options,
        additionalMetadata: options?.additionalMetadata
          ? JSON.stringify(options.additionalMetadata)
          : undefined,
      };
    });

    const limit = 4 * 1024 * 1024; // FIXME configurable GRPC limit

    const batches = batch(workflowRequests, batchSize, limit);

    this.logger.debug(`batching ${batches.length} batches`);

    try {
      const results: WorkflowRunRef<P>[] = [];

      // for loop to ensure serial execution of batches
      for (const { payloads, originalIndices, batchIndex } of batches) {
        // Call the bulk trigger workflow method for this batch
        const bulkTriggerWorkflowResponse = await this.grpc.bulkTriggerWorkflow(
          BulkTriggerWorkflowRequest.create({
            workflows: payloads,
          })
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
            // eslint-disable-next-line no-underscore-dangle
            options?._standaloneTaskName
          );
        });

        results.push(...batchResults);
      }
      return results;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async putRateLimit(key: string, limit: number, duration?: RateLimitDuration) {
    await this.grpc.putRateLimit({
      key,
      limit,
      duration,
    });
  }
}
