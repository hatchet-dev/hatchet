/* eslint-disable no-dupe-class-members */
/* eslint-disable no-underscore-dangle */
import {
  ClientConfig,
  InternalHatchetClient,
  HatchetClientOptions,
  ClientConfigSchema,
} from '@hatchet/clients/hatchet-client';
import { AxiosRequestConfig } from 'axios';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { DurableContext } from '@hatchet/step';
import api, { Api } from '@hatchet/clients/rest';
import { ConfigLoader } from '@hatchet/util/config-loader';
import { DEFAULT_LOGGER } from '@hatchet/clients/hatchet-client/hatchet-logger';
import { z } from 'zod';
import {
  CreateTaskWorkflowOpts,
  CreateWorkflow,
  CreateWorkflowOpts,
  RunOpts,
  BaseWorkflowDeclaration,
  CreateTaskWorkflow,
  WorkflowDeclaration,
  TaskWorkflowDeclaration,
  CreateDurableTaskWorkflow,
} from '../declaration';
import { IHatchetClient } from './client.interface';
import { CreateWorkerOpts, Worker } from './worker';
import { MetricsClient } from './features/metrics';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { RunsClient } from './features/runs';
import { CreateStandaloneDurableTaskOpts } from '../task';
import { InputType, OutputType, UnknownInputType, StrictWorkflowOutputType } from '../types';
import { RatelimitsClient } from './features';

/**
 * HatchetV1 implements the main client interface for interacting with the Hatchet workflow engine.
 * It provides methods for creating and executing workflows, as well as managing workers.
 */
export class HatchetClient implements IHatchetClient {
  /** The underlying v0 client instance */
  _v0: InternalHatchetClient;
  _api: Api;

  /**
   * @deprecated v0 client will be removed in a future release, please upgrade to v1
   */
  get v0() {
    return this._v0;
  }

  /** The tenant ID for the Hatchet client */
  tenantId: string;

  _isV1: boolean | undefined = true;

  get isV1() {
    return true;
  }

  /**
   * Creates a new Hatchet client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   */
  constructor(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ) {
    try {
      const loaded = ConfigLoader.loadClientConfig(config, {
        path: options?.config_path,
      });

      const valid = ClientConfigSchema.parse(loaded);

      let logConstructor = config?.logger;

      if (logConstructor == null) {
        logConstructor = DEFAULT_LOGGER;
      }

      const clientConfig = {
        ...valid,
        logger: logConstructor,
      };

      this.tenantId = clientConfig.tenant_id;
      this._api = api(clientConfig.api_url, clientConfig.token, axiosConfig);
      this._v0 = new InternalHatchetClient(clientConfig, options, axiosConfig, this.runs);
    } catch (e) {
      if (e instanceof z.ZodError) {
        throw new Error(`Invalid client config: ${e.message}`);
      }
      throw e;
    }
  }

  /**
   * Static factory method to create a new Hatchet client instance.
   * @param config - Optional configuration for the client
   * @param options - Optional client options
   * @param axiosConfig - Optional Axios configuration for HTTP requests
   * @returns A new Hatchet client instance
   */
  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): HatchetClient {
    return new HatchetClient(config, options, axiosConfig);
  }

  /**
   * Creates a new workflow definition.
   * @template I - The input type for the workflow
   * @template O - The return type of the workflow
   * @param options - Configuration options for creating the workflow
   * @returns A new Workflow instance
   * @note It is possible to create an orphaned workflow if no client is available using @hatchet/client CreateWorkflow
   */
  workflow<I extends InputType = UnknownInputType, O extends StrictWorkflowOutputType = {}>(
    options: CreateWorkflowOpts
  ): WorkflowDeclaration<I, O> {
    return CreateWorkflow<I, O>(options, this);
  }

  /**
   * Creates a new task workflow.
   * Types can be explicitly specified as generics or inferred from the function signature.
   * @template I The input type for the task
   * @template O The output type of the task
   * @param options Task configuration options
   * @returns A TaskWorkflowDeclaration instance
   */
  task<I extends InputType = UnknownInputType, O extends OutputType = void>(
    options: CreateTaskWorkflowOpts<I, O>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Creates a new task workflow with types inferred from the function parameter.
   * @template Fn The type of the task function with input and output extending JsonObject
   * @param options Task configuration options with function that defines types
   * @returns A TaskWorkflowDeclaration instance with inferred types
   */
  task<
    Fn extends (input: I, ctx?: any) => O | Promise<O>,
    I extends InputType = Parameters<Fn>[0] | UnknownInputType,
    O extends OutputType = ReturnType<Fn> extends Promise<infer P>
      ? P extends OutputType
        ? P
        : void
      : ReturnType<Fn> extends OutputType
        ? ReturnType<Fn>
        : void,
  >(
    options: {
      fn: Fn;
    } & Omit<CreateTaskWorkflowOpts<I, O>, 'fn'>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Implementation of the task method.
   */
  task(options: any): TaskWorkflowDeclaration<any, any> {
    return CreateTaskWorkflow(options, this);
  }

  /**
   * Creates a new durable task workflow.
   * Types can be explicitly specified as generics or inferred from the function signature.
   * @template I The input type for the durable task
   * @template O The output type of the durable task
   * @param options Durable task configuration options
   * @returns A TaskWorkflowDeclaration instance for a durable task
   */
  durableTask<I extends InputType, O extends OutputType>(
    options: CreateStandaloneDurableTaskOpts<I, O>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Creates a new durable task workflow with types inferred from the function parameter.
   * @template Fn The type of the durable task function with input and output extending JsonObject
   * @param options Durable task configuration options with function that defines types
   * @returns A TaskWorkflowDeclaration instance with inferred types
   */
  durableTask<
    Fn extends (input: I, ctx: DurableContext<I>) => O | Promise<O>,
    I extends InputType = Parameters<Fn>[0],
    O extends OutputType = ReturnType<Fn> extends Promise<infer P>
      ? P extends OutputType
        ? P
        : void
      : ReturnType<Fn> extends OutputType
        ? ReturnType<Fn>
        : void,
  >(
    options: {
      fn: Fn;
    } & Omit<CreateStandaloneDurableTaskOpts<I, O>, 'fn'>
  ): TaskWorkflowDeclaration<I, O>;

  /**
   * Implementation of the durableTask method.
   */
  durableTask(options: any): TaskWorkflowDeclaration<any, any> {
    return CreateDurableTaskWorkflow(options, this);
  }

  /**
   * Triggers a workflow run without waiting for completion.
   * @template I - The input type for the workflow
   * @template O - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A WorkflowRunRef containing the run ID and methods to interact with the run
   */
  runNoWait<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: BaseWorkflowDeclaration<I, O> | string | V0Workflow,
    input: I,
    options: RunOpts
  ): WorkflowRunRef<O> {
    let name: string;
    if (typeof workflow === 'string') {
      name = workflow;
    } else if ('id' in workflow) {
      name = workflow.id;
    } else {
      throw new Error('unable to identify workflow');
    }

    return this._v0.admin.runWorkflow<I, O>(name, input, options);
  }

  /**
   * @alias run
   * Triggers a workflow run and waits for the result.
   * @template I - The input type for the workflow
   * @template O - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A promise that resolves with the workflow result
   */
  async runAndWait<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: BaseWorkflowDeclaration<I, O> | string | V0Workflow,
    input: I,
    options: RunOpts = {}
  ): Promise<O> {
    return this.run<I, O>(workflow, input, options);
  }

  /**
   * Triggers a workflow run and waits for the result.
   * @template I - The input type for the workflow
   * @template O - The return type of the workflow
   * @param workflow - The workflow to run, either as a Workflow instance or workflow name
   * @param input - The input data for the workflow
   * @param options - Configuration options for the workflow run
   * @returns A promise that resolves with the workflow result
   */
  async run<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: BaseWorkflowDeclaration<I, O> | string | V0Workflow,
    input: I,
    options: RunOpts = {}
  ): Promise<O> {
    const run = this.runNoWait<I, O>(workflow, input, options);
    return run.output as Promise<O>;
  }

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   */
  get crons() {
    return this._v0.cron;
  }

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   * @deprecated use client.crons instead
   */
  get cron() {
    return this.crons;
  }

  /**
   * Get the schedules client for creating and managing scheduled workflow runs
   * @returns A schedules client instance
   */
  get schedules() {
    return this._v0.schedule;
  }

  /**
   * Get the schedule client for creating and managing scheduled workflow runs
   * @returns A schedule client instance
   * @deprecated use client.schedules instead
   */
  get schedule() {
    return this.schedules;
  }

  /**
   * Get the event client for creating and managing event workflow runs
   * @returns A event client instance
   */
  get events() {
    return this._v0.event;
  }

  /**
   * Get the event client for creating and managing event workflow runs
   * @returns A event client instance
   * @deprecated use client.events instead
   */
  get event() {
    return this.events;
  }

  private _metrics: MetricsClient | undefined;

  /**
   * Get the metrics client for creating and managing metrics
   * @returns A metrics client instance
   */
  get metrics() {
    if (!this._metrics) {
      this._metrics = new MetricsClient(this);
    }
    return this._metrics;
  }

  private _ratelimits: RatelimitsClient | undefined;

  /**
   * Get the rate limits client for creating and managing rate limits
   * @returns A rate limits client instance
   */
  get ratelimits() {
    if (!this._ratelimits) {
      this._ratelimits = new RatelimitsClient(this);
    }
    return this._ratelimits;
  }

  private _runs: RunsClient | undefined;

  /**
   * Get the runs client for creating and managing runs
   * @returns A runs client instance
   */
  get runs() {
    if (!this._runs) {
      this._runs = new RunsClient(this);
    }
    return this._runs;
  }

  private _workflows: WorkflowsClient | undefined;

  /**
   * Get the workflows client for creating and managing workflows
   * @returns A workflows client instance
   */
  get workflows() {
    if (!this._workflows) {
      this._workflows = new WorkflowsClient(this);
    }
    return this._workflows;
  }

  private _workers: WorkersClient | undefined;

  /**
   * Get the workers client for creating and managing workers
   * @returns A workers client instance
   */
  get workers() {
    if (!this._workers) {
      this._workers = new WorkersClient(this);
    }
    return this._workers;
  }

  /**
   * Get the API client for making HTTP requests to the Hatchet API
   * Note: This is not recommended for general use, but is available for advanced scenarios
   * @returns A API client instance
   */
  get api() {
    return this._api;
  }

  /**
   * @deprecated use workflow.run, client.run, or client.* feature methods instead
   */
  get admin() {
    return this._v0.admin;
  }

  /**
   * Creates a new worker instance for processing workflow tasks.
   * @param options - Configuration options for creating the worker
   * @returns A promise that resolves with a new HatchetWorker instance
   */
  worker(name: string, options?: CreateWorkerOpts | number): Promise<Worker> {
    let opts: CreateWorkerOpts = {};
    if (typeof options === 'number') {
      opts = { slots: options };
    } else {
      opts = options || {};
    }

    return Worker.create(this, this._v0, name, opts);
  }

  /**
   * Register a webhook with the worker
   * @param workflows - The workflows to register on the webhooks
   * @returns A promise that resolves when the webhook is registered
   */
  webhooks(workflows: V0Workflow[]) {
    return this._v0.webhooks(workflows);
  }

  runRef<T extends Record<string, any> = any>(id: string): WorkflowRunRef<T> {
    return new WorkflowRunRef<T>(id, this.v0.listener, this.runs);
  }
}
