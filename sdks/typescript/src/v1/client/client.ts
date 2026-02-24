/* eslint-disable no-dupe-class-members */
/* eslint-disable no-underscore-dangle */
import {
  ClientConfig,
  ClientConfigSchema,
  HatchetClientOptions,
  LegacyHatchetClient,
  TaskMiddleware,
  InferMiddlewareBefore,
  InferMiddlewareAfter,
} from '@hatchet/clients/hatchet-client';
import { AxiosRequestConfig } from 'axios';
import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { V0DurableContext } from '@hatchet/step';
import api, { Api } from '@hatchet/clients/rest';
import { ConfigLoader } from '@hatchet/util/config-loader';
import { DEFAULT_LOGGER } from '@hatchet/clients/hatchet-client/hatchet-logger';
import { z } from 'zod';
import { LogLevel } from '@hatchet/clients/event/event-client';
import { RunListenerClient } from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { addTokenMiddleware, channelFactory } from '@hatchet/util/grpc-helpers';
import { createClientFactory } from 'nice-grpc';
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
  CreateDurableTaskWorkflowOpts,
} from '../declaration';
import { IHatchetClient } from './client.interface';
import { CreateWorkerOpts, Worker } from './worker/worker';
import { MetricsClient } from './features/metrics';
import { WorkersClient } from './features/workers';
import { WorkflowsClient } from './features/workflows';
import { RunsClient } from './features/runs';
import {
  InputType,
  OutputType,
  UnknownInputType,
  StrictWorkflowOutputType,
  Resolved,
} from '../types';
import { RatelimitsClient } from './features';
import { AdminClient } from './admin';
import { FiltersClient } from './features/filters';
import { ScheduleClient } from './features/schedules';
import { CronClient } from './features/crons';
import { CELClient } from './features/cel';
import { TenantClient } from './features/tenant';
import { WebhooksClient } from './features/webhooks';

type MergeIfNonEmpty<Base, Extra extends Record<string, any>> = keyof Extra extends never
  ? Base
  : Base & Extra;

/**
 * HatchetV1 implements the main client interface for interacting with the Hatchet workflow engine.
 * It provides methods for creating and executing workflows, as well as managing workers.
 *
 * @template GlobalInput - Global input type required by all tasks. Set via `init<T>()`. Defaults to `{}`.
 * @template MiddlewareBefore - Extra fields merged into task input by pre-middleware hooks. Inferred from middleware config.
 * @template MiddlewareAfter - Extra fields merged into task output by post-middleware hooks. Inferred from middleware config.
 */
export class HatchetClient<
  GlobalInput extends Record<string, any> = {},
  GlobalOutput extends Record<string, any> = {},
  MiddlewareBefore extends Record<string, any> = {},
  MiddlewareAfter extends Record<string, any> = {},
> implements IHatchetClient
{
  /** The underlying v0 client instance */
  _v0: LegacyHatchetClient;
  _api: Api;
  _listener: RunListenerClient;

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

      this._config = clientConfig;

      this.tenantId = clientConfig.tenant_id;
      this._api = api(clientConfig.api_url, clientConfig.token, axiosConfig);

      const clientFactory = createClientFactory().use(addTokenMiddleware(this.config.token));
      const credentials =
        options?.credentials ?? ConfigLoader.createCredentials(this.config.tls_config);

      this._listener = new RunListenerClient(
        this.config,
        channelFactory(this.config, credentials),
        clientFactory,
        this.api
      );

      this._v0 = new LegacyHatchetClient(
        clientConfig,
        options,
        axiosConfig,
        this.runs,
        this._listener
      );
    } catch (e) {
      if (e instanceof z.ZodError) {
        throw new Error(`Invalid client config: ${e.message}`);
      }
      throw e;
    }

    try {
      this.tenant
        .get()
        .then((tenant) => {
          if (tenant.version !== 'V1') {
            this.config
              .logger('client-init', LogLevel.INFO)
              .warn(
                '🚨⚠️‼️ YOU ARE USING A V0 ENGINE WITH A V1 SDK, WHICH IS NOT SUPPORTED. PLEASE UPGRADE YOUR ENGINE TO V1.🚨⚠️‼️'
              );
          }
        })
        .catch(() => {
          // Do nothing here
        });
    } catch (e) {
      // Do nothing here
    }
  }

  /**
   * Static factory method to create a new Hatchet client instance.
   * @template T - Global input type required by all tasks created from this client. Defaults to `{}`.
   * @template U - Global output type required by all tasks created from this client. Defaults to `{}`.
   * @param config - Optional configuration for the client.
   * @param options - Optional client options.
   * @param axiosConfig - Optional Axios configuration for HTTP requests.
   * @returns A new Hatchet client instance. Chain `.withMiddleware()` to attach typed middleware.
   */
  static init<T extends Record<string, any> = {}, U extends Record<string, any> = {}>(
    config?: Omit<Partial<ClientConfig>, 'middleware'>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): HatchetClient<T, U> {
    return new HatchetClient(config, options, axiosConfig) as unknown as HatchetClient<T, U>;
  }

  /**
   * Attaches middleware to this client and returns a re-typed instance
   * with inferred pre/post middleware types.
   *
   * Use this after `init<T, U>()` to get full middleware return-type inference
   * that TypeScript can't provide when global types are explicitly set on `init`.
   */
  withMiddleware<
    const M extends TaskMiddleware<
      Resolved<GlobalInput, MiddlewareBefore>,
      Resolved<GlobalOutput, MiddlewareAfter>
    >,
  >(
    middleware: M
  ): HatchetClient<
    GlobalInput,
    GlobalOutput,
    MiddlewareBefore & InferMiddlewareBefore<M>,
    MiddlewareAfter & InferMiddlewareAfter<M>
  > {
    const existing: TaskMiddleware = (this._config as any).middleware || {};
    const toArray = <T>(v: T | readonly T[] | undefined): T[] => {
      if (v == null) return [];
      if (Array.isArray(v)) return [...v];
      return [v as T];
    };

    (this._config as any).middleware = {
      before: [...toArray(existing.before), ...toArray(middleware.before)],
      after: [...toArray(existing.after), ...toArray(middleware.after)],
    };

    return this as unknown as HatchetClient<
      GlobalInput,
      GlobalOutput,
      MiddlewareBefore & InferMiddlewareBefore<M>,
      MiddlewareAfter & InferMiddlewareAfter<M>
    >;
  }

  private _config: ClientConfig;

  get config() {
    return this._config;
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
  ): WorkflowDeclaration<I, O, Resolved<GlobalInput, MiddlewareBefore>> {
    return CreateWorkflow<I, O>(options, this) as WorkflowDeclaration<
      I,
      O,
      Resolved<GlobalInput, MiddlewareBefore>
    >;
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
    options: CreateTaskWorkflowOpts<
      I & Resolved<GlobalInput, MiddlewareBefore>,
      MergeIfNonEmpty<O, GlobalOutput>
    >
  ): TaskWorkflowDeclaration<I, O, GlobalInput, GlobalOutput, MiddlewareBefore, MiddlewareAfter>;

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
  ): TaskWorkflowDeclaration<I, O, GlobalInput, GlobalOutput, MiddlewareBefore, MiddlewareAfter>;

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
    options: CreateDurableTaskWorkflowOpts<
      I & Resolved<GlobalInput, MiddlewareBefore>,
      MergeIfNonEmpty<O, GlobalOutput>
    >
  ): TaskWorkflowDeclaration<I, O, GlobalInput, GlobalOutput, MiddlewareBefore, MiddlewareAfter>;

  /**
   * Creates a new durable task workflow with types inferred from the function parameter.
   * @template Fn The type of the durable task function with input and output extending JsonObject
   * @param options Durable task configuration options with function that defines types
   * @returns A TaskWorkflowDeclaration instance with inferred types
   */
  durableTask<
    Fn extends (input: I, ctx: V0DurableContext<I>) => O | Promise<O>,
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
    } & Omit<CreateDurableTaskWorkflowOpts<I, O>, 'fn'>
  ): TaskWorkflowDeclaration<I, O, GlobalInput, GlobalOutput, MiddlewareBefore, MiddlewareAfter>;

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
  async runNoWait<I extends InputType = UnknownInputType, O extends OutputType = void>(
    workflow: BaseWorkflowDeclaration<I, O> | string | V0Workflow,
    input: I,
    options: RunOpts
  ): Promise<WorkflowRunRef<O>> {
    let name: string;
    if (typeof workflow === 'string') {
      name = workflow;
    } else if ('id' in workflow) {
      name = workflow.id;
    } else {
      throw new Error('unable to identify workflow');
    }

    return this.admin.runWorkflow<I, O>(name, input, options);
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
    const run = await this.runNoWait<I, O>(workflow, input, options);
    return run.output as Promise<O>;
  }

  private _cel: CELClient | undefined;

  /**
   * Get the CEL client for debugging CEL expressions
   * @returns A CEL client instance
   */
  get cel() {
    if (!this._cel) {
      this._cel = new CELClient(this);
    }
    return this._cel;
  }

  private _crons: CronClient | undefined;

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   */
  get crons() {
    if (!this._crons) {
      this._crons = new CronClient(this);
    }
    return this._crons;
  }

  /**
   * Get the cron client for creating and managing cron workflow runs
   * @returns A cron client instance
   * @deprecated use client.crons instead
   */
  get cron() {
    return this.crons;
  }

  private _scheduled: ScheduleClient | undefined;

  /**
   * Get the schedules client for creating and managing scheduled workflow runs
   * @returns A schedules client instance
   */
  get scheduled() {
    if (!this._scheduled) {
      this._scheduled = new ScheduleClient(this);
    }
    return this._scheduled;
  }

  /**
   * Get the schedule client for creating and managing scheduled workflow runs
   * @returns A schedule client instance
   * @deprecated use client.scheduled instead
   */
  get schedule() {
    return this.scheduled;
  }

  /**
   * @alias scheduled
   */
  get schedules() {
    return this.scheduled;
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

  private _filters: FiltersClient | undefined;

  /**
   * Get the filters client for creating and managing filters
   * @returns A filters client instance
   */
  get filters() {
    if (!this._filters) {
      this._filters = new FiltersClient(this);
    }
    return this._filters;
  }

  private _tenant: TenantClient | undefined;
  /**
   * Get the tenant client for managing tenants
   * @returns A tenant client instance
   */
  get tenant() {
    if (!this._tenant) {
      this._tenant = new TenantClient(this);
    }
    return this._tenant;
  }

  private _webhooks: WebhooksClient | undefined;

  /**
   * Get the webhooks client for creating and managing webhooks
   * @returns A webhooks client instance
   */
  get webhooks() {
    if (!this._webhooks) {
      this._webhooks = new WebhooksClient(this);
    }
    return this._webhooks;
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

  /**
   * Get the tasks client for creating and managing tasks
   * @returns A tasks client instance
   */
  get tasks() {
    return this.workflows;
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

  _admin: AdminClient | undefined;

  /**
   * Get the admin client for creating and managing workflows
   * @returns A admin client instance
   */
  get admin() {
    if (!this._admin) {
      this._admin = new AdminClient(this._v0.config, this.api, this.runs);
    }
    return this._admin;
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
  v0webhooks(workflows: V0Workflow[]) {
    return this._v0.webhooks(workflows);
  }

  runRef<T extends Record<string, any> = any>(id: string): WorkflowRunRef<T> {
    return this.runs.runRef(id);
  }
}
