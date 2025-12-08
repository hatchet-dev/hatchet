/* eslint-disable no-underscore-dangle */
/* eslint-disable no-nested-ternary */
import HatchetError from '@util/errors/hatchet-error';
import { Action, ActionListener } from '@clients/dispatcher/action-listener';
import {
  StepActionEvent,
  StepActionEventType,
  ActionType,
  GroupKeyActionEvent,
  GroupKeyActionEventType,
  actionTypeFromJSON,
} from '@hatchet/protoc/dispatcher';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';
import { Workflow } from '@hatchet/workflow';
import {
  ConcurrencyLimitStrategy,
  CreateWorkflowJobOpts,
  CreateWorkflowStepOpts,
  DesiredWorkerLabels,
  WorkflowConcurrencyOpts,
} from '@hatchet/protoc/workflows';
import { Logger } from '@hatchet/util/logger';
import { WebhookWorkerCreateRequest } from '@clients/rest/generated/data-contracts';
import { BaseWorkflowDeclaration, WorkflowDefinition, HatchetClient } from '@hatchet/v1';
import { CreateTaskOpts } from '@hatchet/protoc/v1/workflows';
import {
  CreateOnFailureTaskOpts,
  CreateOnSuccessTaskOpts,
  CreateWorkflowDurableTaskOpts,
  CreateWorkflowTaskOpts,
  BatchTaskConfig,
  NonRetryableError,
} from '@hatchet/v1/task';
import { taskConditionsToPb } from '@hatchet/v1/conditions/transformer';

import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { CreateStep, mapRateLimit, StepRunFunction } from '@hatchet/step';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { Context, DurableContext } from './context';
import { parentRunContextManager } from '../../parent-run-context-vars';
import { HealthServer, workerStatus, type WorkerStatus } from './health-server';

export type ActionRegistry = Record<Action['actionId'], Function>;

type BatchQueueItem = {
  ctx: Context<any, any>;
  resolve: (value: any) => void;
  reject: (reason?: any) => void;
};

type BatchState = {
  expectedSize: number;
  items: Map<number, BatchQueueItem>;
  started: boolean;
  startReason?: string;
  triggerTime?: Date;
  batchKey?: string;
};

type BatchController = {
  enqueueItem: (metadata: {
    batchId: string;
    expectedSize: number;
    index: number;
    ctx: Context<any, any>;
    resolve: (value: any) => void;
    reject: (reason?: any) => void;
    batchKey?: string;
  }) => void;
  startBatch: (metadata: {
    batchId: string;
    expectedSize: number;
    triggerReason?: string;
    triggerTime?: Date;
    batchKey?: string;
  }) => void;
};

export interface WorkerOpts {
  name: string;
  handleKill?: boolean;
  maxRuns?: number;
  labels?: WorkerLabels;
  healthPort?: number;
  enableHealthServer?: boolean;
}

export class V1Worker {
  client: HatchetClient;
  name: string;
  workerId: string | undefined;
  killing: boolean;
  handle_kill: boolean;

  action_registry: ActionRegistry;
  workflow_registry: Array<WorkflowDefinition | Workflow> = [];
  listener: ActionListener | undefined;
  futures: Record<Action['stepRunId'], HatchetPromise<any>> = {};
  contexts: Record<Action['stepRunId'], Context<any, any>> = {};
  maxRuns?: number;

  logger: Logger;

  registeredWorkflowPromises: Array<Promise<any>> = [];

  labels: WorkerLabels = {};

  healthPort: number;
  enableHealthServer: boolean;

  private healthServer: HealthServer | undefined;
  private status: WorkerStatus = workerStatus.INITIALIZED;
  private batchControllers: Map<string, BatchController> = new Map();

  constructor(
    client: HatchetClient,
    options: {
      name: string;
      handleKill?: boolean;
      maxRuns?: number;
      labels?: WorkerLabels;
    }
  ) {
    this.client = client;
    this.name = applyNamespace(options.name, this.client.config.namespace);
    this.action_registry = {};
    this.maxRuns = options.maxRuns;

    this.labels = options.labels || {};

    this.enableHealthServer = client.config.healthcheck?.enabled ?? false;
    this.healthPort = client.config.healthcheck?.port ?? 8001;

    process.on('SIGTERM', () => this.exitGracefully(true));
    process.on('SIGINT', () => this.exitGracefully(true));

    this.killing = false;
    this.handle_kill = options.handleKill === undefined ? true : options.handleKill;

    this.logger = client.config.logger(`Worker/${this.name}`, this.client.config.log_level);

    if (this.enableHealthServer && this.healthPort) {
      this.initializeHealthServer();
    }
  }

  private handleStartBatchAction(action: Action) {
    const { actionId } = action;
    const { batchId } = action;
    const { batchStart } = action;

    if (!actionId) {
      this.logger.error('Received start batch action without action id');
      return;
    }

    if (!batchId) {
      this.logger.error('Received start batch action without batch id');
      return;
    }

    if (!batchStart) {
      this.logger.error(`Received start batch action without payload for batch ${batchId}`);
      return;
    }

    const controller = this.batchControllers.get(actionId);

    if (!controller) {
      this.logger.error(
        `No batch controller registered for action '${actionId}', cannot start batch ${batchId}`
      );
      return;
    }

    controller.startBatch({
      batchId,
      expectedSize: batchStart.expectedSize ?? 0,
      triggerReason: batchStart.triggerReason ?? undefined,
      triggerTime: batchStart.triggerTime ?? undefined,
      batchKey: batchStart.batchKey ?? action.batchKey ?? undefined,
    });
  }

  private initializeHealthServer(): void {
    if (!this.healthPort) {
      this.logger.warn('Health server enabled but no port specified');
      return;
    }

    this.healthServer = new HealthServer(
      this.healthPort,
      () => this.status,
      this.name,
      () => this.getAvailableSlots(),
      () => this.getRegisteredActions(),
      () => this.getFilteredLabels(),
      this.logger
    );
  }

  private getAvailableSlots(): number {
    if (!this.maxRuns) {
      return 0;
    }
    const currentRuns = Object.keys(this.futures).length;
    return Math.max(0, this.maxRuns - currentRuns);
  }

  private getRegisteredActions(): string[] {
    return Object.keys(this.action_registry);
  }

  private getFilteredLabels(): Record<string, string | number> {
    const filtered: Record<string, string | number> = {};
    for (const [key, value] of Object.entries(this.labels)) {
      if (value !== undefined) {
        filtered[key] = value;
      }
    }
    return filtered;
  }

  private setStatus(status: WorkerStatus): void {
    this.status = status;
    this.logger.debug(`Worker status changed to: ${status}`);
  }

  private registerActions(workflow: Workflow) {
    const newActions = workflow.steps.reduce<ActionRegistry>((acc, step) => {
      acc[`${workflow.id}:${step.name.toLowerCase()}`] = step.run;
      return acc;
    }, {});

    const onFailureAction = workflow.onFailure
      ? {
          [`${workflow.id}-on-failure:${workflow.onFailure.name}`]: workflow.onFailure.run,
        }
      : {};

    this.action_registry = {
      ...this.action_registry,
      ...newActions,
      ...onFailureAction,
    };

    this.action_registry =
      workflow.concurrency?.name && workflow.concurrency.key
        ? {
            ...this.action_registry,
            [`${workflow.id}:${workflow.concurrency.name.toLowerCase()}`]: workflow.concurrency.key,
          }
        : {
            ...this.action_registry,
          };
  }

  getHandler(workflows: Workflow[]) {
    throw new Error('Not implemented');
    // TODO v1
    // for (const workflow of workflows) {
    //   const wf: Workflow = {
    //     ...workflow,
    //     id: this.client.config.namespace + workflow.id,
    //   };

    //   this.registerActions(wf);
    // }

    // return new WebhookHandler(this, workflows);
  }

  async registerWebhook(webhook: WebhookWorkerCreateRequest) {
    return this.client._v0.admin.registerWebhook({ ...webhook });
  }

  /**
   * @deprecated use registerWorkflow instead
   */
  async register_workflow(initWorkflow: Workflow) {
    return this.registerWorkflow(initWorkflow);
  }

  registerDurableActionsV1(workflow: WorkflowDefinition) {
    const newActions = workflow._durableTasks
      .filter((task) => !!task.fn)
      .reduce<ActionRegistry>((acc, task) => {
        acc[
          `${applyNamespace(
            workflow.name,
            this.client.config.namespace
          ).toLowerCase()}:${task.name.toLowerCase()}`
        ] = (ctx: Context<any, any>) => task.fn!(ctx.input, ctx as DurableContext<any, any>);
        return acc;
      }, {});

    this.action_registry = {
      ...this.action_registry,
      ...newActions,
    };
  }

  private createBatchActionHandler(
    actionId: string,
    workflow: WorkflowDefinition,
    task: CreateWorkflowTaskOpts<any, any>
  ) {
    const batchedTask = task as CreateWorkflowTaskOpts<any, any> & {
      batch?: BatchTaskConfig<any, any>;
    };
    const { batch } = batchedTask;

    if (!batch) {
      throw new HatchetError(`Batch configuration missing for task '${task.name}'`);
    }

    const { batchSize } = batch;

    if (!Number.isInteger(batchSize) || batchSize <= 0) {
      throw new HatchetError(
        `Batch task '${workflow.name}:${task.name}' must have a positive integer batchSize`
      );
    }

    const batches = new Map<string, BatchState>();

    const flushBatch = async (batchId: string, state: BatchState) => {
      const orderedItems = Array.from(state.items.entries())
        .sort(([a], [b]) => a - b)
        .map(([, item]) => item);
      const contexts = orderedItems.map((item) => item.ctx);
      const inputs = contexts.map((ctx) => ctx.input);

      try {
        const outputs = await batch.fn(inputs, contexts);

        if (!Array.isArray(outputs)) {
          throw new HatchetError(`Batch task '${task.name}' must return an array of outputs`);
        }

        if (outputs.length !== orderedItems.length) {
          throw new HatchetError(
            `Batch task '${task.name}' returned ${outputs.length} outputs for ${orderedItems.length} inputs`
          );
        }

        outputs.forEach((output, index) => orderedItems[index].resolve(output));
      } catch (err: any) {
        const error = err instanceof Error ? err : new Error(String(err));
        orderedItems.forEach(({ reject }) => reject(error));
        this.logger.error(
          `Batch task '${workflow.name}:${task.name}' failed to process batch ${batchId}`,
          error
        );
      }
    };

    const maybeFlush = (batchId: string) => {
      const state = batches.get(batchId);

      if (!state || !state.started) {
        return;
      }

      if (state.items.size !== state.expectedSize) {
        return;
      }

      batches.delete(batchId);

      flushBatch(batchId, state).catch((err: unknown) => {
        const error = err instanceof Error ? err : new Error(String(err));
        this.logger.error(
          `Batch task '${workflow.name}:${task.name}' flush encountered an error`,
          error
        );
      });
    };

    const controller: BatchController = {
      enqueueItem: ({ batchId, expectedSize, index, ctx, resolve, reject, batchKey }) => {
        let state = batches.get(batchId);

        if (!state) {
          state = {
            expectedSize,
            items: new Map<number, BatchQueueItem>(),
            started: false,
            batchKey,
          };
          batches.set(batchId, state);
        } else if (state.expectedSize !== expectedSize) {
          this.logger.error(
            `Batch metadata mismatch for '${workflow.name}:${task.name}' batch ${batchId}: expected size ${state.expectedSize}, received ${expectedSize}`
          );
          reject(
            new HatchetError(
              `Batch metadata mismatch for task '${workflow.name}:${task.name}' (batch ${batchId})`
            )
          );
          return;
        }

        if (state.items.has(index)) {
          this.logger.error(
            `Duplicate batch index ${index} for batch ${batchId} on task '${workflow.name}:${task.name}'`
          );
          reject(
            new HatchetError(
              `Duplicate batch index ${index} for task '${workflow.name}:${task.name}' (batch ${batchId})`
            )
          );
          return;
        }

        state.items.set(index, { ctx, resolve, reject });

        if (batchKey && !state.batchKey) {
          state.batchKey = batchKey;
        }

        if (state.started) {
          maybeFlush(batchId);
        }
      },
      startBatch: ({ batchId, expectedSize, triggerReason, triggerTime, batchKey }) => {
        let state = batches.get(batchId);

        if (!state) {
          state = {
            expectedSize: expectedSize > 0 ? expectedSize : batchSize,
            items: new Map<number, BatchQueueItem>(),
            started: true,
            batchKey,
          };
          batches.set(batchId, state);
        } else {
          if (expectedSize > 0 && state.expectedSize !== expectedSize) {
            this.logger.warn(
              `Batch metadata mismatch for '${workflow.name}:${task.name}' batch ${batchId}: expected size ${state.expectedSize}, received start signal with ${expectedSize}`
            );
            state.expectedSize = expectedSize;
          }

          state.started = true;
        }

        state.startReason = triggerReason;
        state.triggerTime = triggerTime;
        if (batchKey) {
          state.batchKey = batchKey;
        }

        maybeFlush(batchId);
      },
    };

    this.batchControllers.set(actionId, controller);

    return (ctx: Context<any, any>) =>
      new Promise((resolve, reject) => {
        const { batchId, batchSize: expectedSize, batchIndex } = ctx.action;

        if (!batchId || expectedSize === undefined || batchIndex === undefined) {
          reject(
            new HatchetError(
              `Batch metadata missing for task '${workflow.name}:${task.name}', cannot process batch`
            )
          );
          return;
        }

        if (expectedSize <= 0) {
          reject(
            new HatchetError(
              `Batch task '${workflow.name}:${task.name}' received invalid batch size: ${expectedSize}`
            )
          );
          return;
        }

        const index = Number(batchIndex);

        if (!Number.isInteger(index) || index < 0) {
          reject(
            new HatchetError(
              `Batch task '${workflow.name}:${task.name}' received invalid batch index: ${batchIndex}`
            )
          );
          return;
        }

        controller.enqueueItem({
          batchId,
          expectedSize,
          index,
          ctx,
          resolve,
          reject,
          batchKey: ctx.action.batchKey ?? undefined,
        });
      });
  }

  private registerActionsV1(workflow: WorkflowDefinition) {
    const newActions = workflow._tasks.reduce<ActionRegistry>((acc, task) => {
      const actionId = `${workflow.name}:${task.name.toLowerCase()}`;

      const batchedTask = task as CreateWorkflowTaskOpts<any, any> & {
        batch?: BatchTaskConfig<any, any>;
      };

      if (batchedTask.batch) {
        acc[actionId] = this.createBatchActionHandler(actionId, workflow, batchedTask);
      } else if (task.fn) {
        acc[actionId] = (ctx: Context<any, any>) => task.fn!(ctx.input, ctx);
      }

      return acc;
    }, {});

    const onFailureFn = workflow.onFailure
      ? typeof workflow.onFailure === 'function'
        ? workflow.onFailure
        : workflow.onFailure.fn
      : undefined;

    const onFailureAction = onFailureFn
      ? {
          [onFailureTaskName(workflow)]: (ctx: Context<any, any>) => onFailureFn(ctx.input, ctx),
        }
      : {};

    this.action_registry = {
      ...this.action_registry,
      ...newActions,
      ...onFailureAction,
    };
  }

  async registerWorkflowV1(
    initWorkflow: BaseWorkflowDeclaration<any, any>,
    durable: boolean = false
  ) {
    // patch the namespace
    const workflow: WorkflowDefinition = {
      ...initWorkflow.definition,
      name: applyNamespace(
        initWorkflow.definition.name,
        this.client.config.namespace
      ).toLowerCase(),
    };

    try {
      const { concurrency } = workflow;

      let onFailureTask: CreateTaskOpts | undefined;

      if (workflow.onFailure && typeof workflow.onFailure === 'function') {
        onFailureTask = {
          readableId: 'on-failure-task',
          action: onFailureTaskName(workflow),
          timeout: '60s',
          inputs: '{}',
          parents: [],
          retries: 0,
          rateLimits: [],
          workerLabels: {},
          concurrency: [],
        };
      }

      if (workflow.onFailure && typeof workflow.onFailure === 'object') {
        const onFailure = workflow.onFailure as CreateOnFailureTaskOpts<any, any>;

        onFailureTask = {
          readableId: 'on-failure-task',
          action: onFailureTaskName(workflow),
          timeout: onFailure.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s',
          scheduleTimeout: onFailure.scheduleTimeout || workflow.taskDefaults?.scheduleTimeout,
          inputs: '{}',
          parents: [],
          retries: onFailure.retries || workflow.taskDefaults?.retries || 0,
          rateLimits: mapRateLimit(onFailure.rateLimits || workflow.taskDefaults?.rateLimits),
          workerLabels: toPbWorkerLabel(
            onFailure.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
          ),
          concurrency: [],
          backoffFactor: onFailure.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
          backoffMaxSeconds:
            onFailure.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
        };
      }

      let onSuccessTask: CreateWorkflowTaskOpts<any, any> | undefined;

      if (!durable && workflow.onSuccess && typeof workflow.onSuccess === 'function') {
        const parents = getLeaves([...workflow._tasks, ...workflow._durableTasks]);

        onSuccessTask = {
          name: 'on-success-task',
          fn: workflow.onSuccess,
          timeout: '60s',
          parents,
          retries: 0,
          rateLimits: [],
          desiredWorkerLabels: {},
          concurrency: [],
        };
      }

      if (!durable && workflow.onSuccess && typeof workflow.onSuccess === 'object') {
        const onSuccess = workflow.onSuccess as CreateOnSuccessTaskOpts<any, any>;
        const parents = getLeaves([...workflow._tasks, ...workflow._durableTasks]);

        onSuccessTask = {
          name: 'on-success-task',
          fn: onSuccess.fn,
          timeout: onSuccess.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s',
          scheduleTimeout: onSuccess.scheduleTimeout || workflow.taskDefaults?.scheduleTimeout,
          parents,
          retries: onSuccess.retries || workflow.taskDefaults?.retries || 0,
          rateLimits: onSuccess.rateLimits || workflow.taskDefaults?.rateLimits,
          desiredWorkerLabels: onSuccess.desiredWorkerLabels || workflow.taskDefaults?.workerLabels,
          concurrency: onSuccess.concurrency || workflow.taskDefaults?.concurrency,
          backoff: onSuccess.backoff || workflow.taskDefaults?.backoff,
        };
      }

      if (onSuccessTask) {
        workflow._tasks.push(onSuccessTask);
      }

      // cron and event triggers
      if (workflow.on) {
        this.logger.warn(
          `\`on\` for event and cron triggers is deprecated and will be removed soon, use \`onEvents\` and \`onCrons\` instead for ${
            workflow.name
          }`
        );
      }

      const eventTriggers = [
        ...(workflow.onEvents || []).map((event) =>
          applyNamespace(event, this.client.config.namespace)
        ),
        ...(workflow.on?.event
          ? [applyNamespace(workflow.on.event, this.client.config.namespace)]
          : []),
      ];
      const cronTriggers = [
        ...(workflow.onCrons || []),
        ...(workflow.on?.cron ? [workflow.on.cron] : []),
      ];

      const concurrencyArr = Array.isArray(concurrency) ? concurrency : [];
      const concurrencySolo = !Array.isArray(concurrency) ? concurrency : undefined;

      const registeredWorkflow = this.client._v0.admin.putWorkflowV1({
        name: workflow.name,
        description: workflow.description || '',
        version: workflow.version || '',
        eventTriggers,
        cronTriggers,
        sticky: workflow.sticky,
        concurrencyArr,
        onFailureTask,
        defaultPriority: workflow.defaultPriority,
        tasks: [...workflow._tasks, ...workflow._durableTasks].map<CreateTaskOpts>((task) => {
          const batchedTask = task as CreateWorkflowTaskOpts<any, any> & {
            batch?: BatchTaskConfig<any, any>;
          };

          return {
            readableId: task.name,
            action: `${workflow.name}:${task.name}`,
            timeout:
              task.executionTimeout ||
              task.timeout ||
              workflow.taskDefaults?.executionTimeout ||
              '60s',
            scheduleTimeout: task.scheduleTimeout || workflow.taskDefaults?.scheduleTimeout,
            inputs: '{}',
            parents: task.parents?.map((p) => p.name) ?? [],
            userData: '{}',
            retries: task.retries || workflow.taskDefaults?.retries || 0,
            rateLimits: mapRateLimit(task.rateLimits || workflow.taskDefaults?.rateLimits),
            workerLabels: toPbWorkerLabel(
              task.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
            ),
            backoffFactor: task.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
            backoffMaxSeconds:
              task.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
            conditions: taskConditionsToPb(task),
            concurrency: task.concurrency
              ? Array.isArray(task.concurrency)
                ? task.concurrency
                : [task.concurrency]
              : workflow.taskDefaults?.concurrency
                ? Array.isArray(workflow.taskDefaults.concurrency)
                  ? workflow.taskDefaults.concurrency
                  : [workflow.taskDefaults.concurrency]
                : [],
            batch: batchedTask.batch
              ? {
                  batchSize: batchedTask.batch.batchSize,
                  flushIntervalMs: batchedTask.batch.flushInterval,
                  batchKey: batchedTask.batch.batchKey,
                  maxRuns:
                    typeof batchedTask.batch.maxRuns === 'number'
                      ? batchedTask.batch.maxRuns
                      : undefined,
                }
              : undefined,
          };
        }),
        concurrency: concurrencySolo,
        defaultFilters:
          workflow.defaultFilters?.map((f) => ({
            scope: f.scope,
            expression: f.expression,
            payload: f.payload ? new TextEncoder().encode(JSON.stringify(f.payload)) : undefined,
          })) ?? [],
      });
      this.registeredWorkflowPromises.push(registeredWorkflow);
      await registeredWorkflow;
      this.workflow_registry.push(workflow);
    } catch (e: any) {
      throw new HatchetError(`Could not register workflow: ${e.message}`);
    }

    this.registerActionsV1(workflow);
  }

  async registerWorkflow(initWorkflow: Workflow) {
    const workflow: Workflow = {
      ...initWorkflow,
      id: applyNamespace(initWorkflow.id, this.client.config.namespace).toLowerCase(),
    };
    try {
      if (workflow.concurrency?.key && workflow.concurrency.expression) {
        throw new HatchetError(
          'Cannot have both key function and expression in workflow concurrency configuration'
        );
      }

      const concurrency: WorkflowConcurrencyOpts | undefined =
        workflow.concurrency?.name || workflow.concurrency?.expression
          ? {
              action: !workflow.concurrency.expression
                ? `${workflow.id}:${workflow.concurrency.name}`
                : undefined,
              maxRuns: workflow.concurrency.maxRuns || 1,
              expression: workflow.concurrency.expression,
              limitStrategy:
                workflow.concurrency.limitStrategy || ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
            }
          : undefined;

      const onFailureJob: CreateWorkflowJobOpts | undefined = workflow.onFailure
        ? {
            name: `${workflow.id}-on-failure`,
            description: workflow.description,
            steps: [
              {
                readableId: workflow.onFailure.name,
                action: `${workflow.id}-on-failure:${workflow.onFailure.name}`,
                timeout: workflow.onFailure.timeout || '60s',
                inputs: '{}',
                parents: [],
                userData: '{}',
                retries: workflow.onFailure.retries || 0,
                rateLimits: mapRateLimit(workflow.onFailure.rate_limits),
                workerLabels: {}, // no worker labels for on failure steps
              },
            ],
          }
        : undefined;

      const registeredWorkflow = this.client._v0.admin.putWorkflow({
        name: workflow.id,
        description: workflow.description,
        version: workflow.version || '',
        eventTriggers:
          workflow.on && workflow.on.event
            ? [applyNamespace(workflow.on.event, this.client.config.namespace)]
            : [],
        cronTriggers: workflow.on && workflow.on.cron ? [workflow.on.cron] : [],
        scheduledTriggers: [],
        concurrency,
        scheduleTimeout: workflow.scheduleTimeout,
        onFailureJob,
        sticky: workflow.sticky,
        jobs: [
          {
            name: workflow.id,
            description: workflow.description,
            steps: workflow.steps.map<CreateWorkflowStepOpts>((step) => ({
              readableId: step.name,
              action: `${workflow.id}:${step.name}`,
              timeout: step.timeout || '60s',
              inputs: '{}',
              parents: step.parents ?? [],
              userData: '{}',
              retries: step.retries || 0,
              rateLimits: mapRateLimit(step.rate_limits),
              workerLabels: toPbWorkerLabel(step.worker_labels),
              backoffFactor: step.backoff?.factor,
              backoffMaxSeconds: step.backoff?.maxSeconds,
            })),
          },
        ],
      });
      this.registeredWorkflowPromises.push(registeredWorkflow);
      await registeredWorkflow;
      this.workflow_registry.push(workflow);
    } catch (e: any) {
      throw new HatchetError(`Could not register workflow: ${e.message}`);
    }

    this.registerActions(workflow);
  }

  registerAction<T, K>(actionId: string, action: StepRunFunction<T, K>) {
    this.action_registry[actionId.toLowerCase()] = action;
  }

  async handleStartStepRun(action: Action) {
    const { actionId } = action;

    try {
      // Note: we always use a DurableContext since its a superset of the Context class
      const context = new DurableContext(action, this.client, this);
      this.contexts[action.stepRunId] = context;

      const step = this.action_registry[actionId];

      if (!step) {
        this.logger.error(`Registered actions: '${Object.keys(this.action_registry).join(', ')}'`);
        this.logger.error(`Could not find step '${actionId}'`);
        return;
      }

      const run = async () => {
        parentRunContextManager.setContext({
          parentId: action.workflowRunId,
          parentRunId: action.stepRunId,
          childIndex: 0,
          desiredWorkerId: this.workerId || '',
        });
        return step(context);
      };

      const success = async (result: any) => {
        this.logger.info(`Step run ${action.stepRunId} succeeded`);

        try {
          // Send the action event to the dispatcher
          const event = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
            false,
            result || null,
            action.retryCount
          );
          await this.client._v0.dispatcher.sendStepActionEvent(event);
        } catch (actionEventError: any) {
          this.logger.error(
            `Could not send completed action event: ${actionEventError.message || actionEventError}`
          );

          // send a failure event
          const failureEvent = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_FAILED,
            false,
            actionEventError.message,
            action.retryCount
          );

          try {
            await this.client._v0.dispatcher.sendStepActionEvent(failureEvent);
          } catch (failureEventError: any) {
            this.logger.error(
              `Could not send failed action event: ${failureEventError.message || failureEventError}`
            );
          }

          this.logger.error(
            `Could not send action event: ${actionEventError.message || actionEventError}`
          );
        } finally {
          // delete the run from the futures
          delete this.futures[action.stepRunId];
          delete this.contexts[action.stepRunId];
        }
      };

      const failure = async (error: any) => {
        this.logger.error(`Step run ${action.stepRunId} failed: ${error.message}`);

        if (error.stack) {
          this.logger.error(error.stack);
        }

        const shouldNotRetry = error instanceof NonRetryableError;

        try {
          // Send the action event to the dispatcher
          const event = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_FAILED,
            shouldNotRetry,
            {
              message: error?.message,
              stack: error?.stack,
            },
            action.retryCount
          );
          await this.client._v0.dispatcher.sendStepActionEvent(event);
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        } finally {
          // delete the run from the futures
          delete this.futures[action.stepRunId];
          delete this.contexts[action.stepRunId];
        }
      };

      const future = new HatchetPromise(
        (async () => {
          let result: any;
          try {
            result = await run();
          } catch (e: any) {
            await failure(e);
            return;
          }
          await success(result);
        })()
      );
      this.futures[action.stepRunId] = future;

      // Send the action event to the dispatcher
      const event = this.getStepActionEvent(
        action,
        StepActionEventType.STEP_EVENT_TYPE_STARTED,
        false,
        undefined,
        action.retryCount
      );
      this.client._v0.dispatcher.sendStepActionEvent(event).catch((e) => {
        this.logger.error(`Could not send action event: ${e.message}`);
      });

      try {
        await future.promise;
      } catch (e: any) {
        this.logger.error('Could not wait for step run to finish: ', e);
      }
    } catch (e: any) {
      this.logger.error('Could not send action event (outer): ', e);
    }
  }

  async handleStartGroupKeyRun(action: Action) {
    const { actionId } = action;

    this.logger.error(
      'Concurrency Key Functions have been deprecated and will be removed in a future release. Use Concurrency Expressions instead.'
    );

    try {
      const context = new Context(action, this.client, this);

      const key = action.getGroupKeyRunId;

      if (!key) {
        this.logger.error(`No group key run id provided for action ${actionId}`);
        return;
      }

      this.contexts[key] = context;

      this.logger.debug(`Starting group key run ${key}`);

      const step = this.action_registry[actionId];

      if (!step) {
        this.logger.error(`Could not find step '${actionId}'`);
        return;
      }

      const run = async () => {
        return step(context);
      };

      const success = (result: any) => {
        this.logger.info(`Step run ${action.stepRunId} succeeded`);

        try {
          // Send the action event to the dispatcher
          const event = this.getGroupKeyActionEvent(
            action,
            GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_COMPLETED,
            result
          );
          this.client._v0.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
            this.logger.error(`Could not send action event: ${e.message}`);
          });
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        } finally {
          // delete the run from the futures
          delete this.futures[key];
          delete this.contexts[key];
        }
      };

      const failure = (error: any) => {
        this.logger.error(`Step run ${key} failed: ${error.message}`);

        try {
          // Send the action event to the dispatcher
          const event = this.getGroupKeyActionEvent(
            action,
            GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_FAILED,
            error
          );
          this.client._v0.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
            this.logger.error(`Could not send action event: ${e.message}`);
          });
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        } finally {
          // delete the run from the futures
          delete this.futures[key];
          delete this.contexts[key];
        }
      };

      const future = new HatchetPromise(run().then(success).catch(failure));
      this.futures[key] = future;

      // Send the action event to the dispatcher
      const event = this.getGroupKeyActionEvent(
        action,
        GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_STARTED
      );
      this.client._v0.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
        this.logger.error(`Could not send action event: ${e.message}`);
      });

      await future.promise;
    } catch (e: any) {
      this.logger.error(`Could not send action event: ${e.message}`);
    }
  }

  getStepActionEvent(
    action: Action,
    eventType: StepActionEventType,
    shouldNotRetry: boolean,
    payload: any = '',
    retryCount: number = 0
  ): StepActionEvent {
    return {
      workerId: this.name,
      jobId: action.jobId,
      jobRunId: action.jobRunId,
      stepId: action.stepId,
      stepRunId: action.stepRunId,
      actionId: action.actionId,
      eventTimestamp: new Date(),
      eventType,
      eventPayload: JSON.stringify(payload),
      shouldNotRetry,
      retryCount,
    };
  }

  getGroupKeyActionEvent(
    action: Action,
    eventType: GroupKeyActionEventType,
    payload: any = ''
  ): GroupKeyActionEvent {
    if (!action.getGroupKeyRunId) {
      throw new HatchetError('No group key run id provided');
    }
    return {
      workerId: this.name,
      workflowRunId: action.workflowRunId,
      getGroupKeyRunId: action.getGroupKeyRunId,
      actionId: action.actionId,
      eventTimestamp: new Date(),
      eventType,
      eventPayload: JSON.stringify(payload),
    };
  }

  async handleCancelStepRun(action: Action) {
    const { stepRunId } = action;
    try {
      this.logger.info(`Cancelling step run ${action.stepRunId}`);
      const future = this.futures[stepRunId];
      const context = this.contexts[stepRunId];

      if (context && context.abortController) {
        context.abortController.abort('Cancelled by worker');
      }

      if (future) {
        future.promise.catch(() => {
          this.logger.info(`Cancelled step run ${action.stepRunId}`);
        });
        future.cancel('Cancelled by worker');
        await future.promise;
      }
    } catch (e: any) {
      this.logger.error('Could not cancel step run: ', e);
    } finally {
      delete this.futures[stepRunId];
      delete this.contexts[stepRunId];
    }
  }

  async stop() {
    await this.exitGracefully(false);
  }

  async exitGracefully(handleKill: boolean) {
    this.killing = true;
    this.setStatus(workerStatus.UNHEALTHY);

    this.logger.info('Starting to exit...');

    try {
      await this.listener?.unregister();
    } catch (e: any) {
      this.logger.error(`Could not unregister listener: ${e.message}`);
    }

    this.logger.info('Gracefully exiting hatchet worker, running tasks will attempt to finish...');

    // attempt to wait for futures to finish
    await Promise.all(Object.values(this.futures).map(({ promise }) => promise));

    this.logger.info('Successfully finished pending tasks.');

    if (this.healthServer) {
      try {
        await this.healthServer.stop();
      } catch (e: any) {
        this.logger.error(`Could not stop health server: ${e.message}`);
      }
    }

    if (handleKill) {
      this.logger.info('Exiting hatchet worker...');
      process.exit(0);
    }
  }

  async start() {
    this.setStatus(workerStatus.STARTING);

    if (this.healthServer) {
      try {
        await this.healthServer.start();
      } catch (e: any) {
        this.logger.error(`Could not start health server: ${e.message}`);
        this.setStatus(workerStatus.UNHEALTHY);
        return;
      }
    }

    // ensure all workflows are registered
    await Promise.all(this.registeredWorkflowPromises);

    if (Object.keys(this.action_registry).length === 0) {
      return;
    }

    try {
      this.listener = await this.client._v0.dispatcher.getActionListener({
        workerName: this.name,
        services: ['default'],
        actions: Object.keys(this.action_registry),
        maxRuns: this.maxRuns,
        labels: this.labels,
      });

      this.workerId = this.listener.workerId;
      this.setStatus(workerStatus.HEALTHY);

      const generator = this.listener.actions();

      this.logger.info(`Worker ${this.name} listening for actions`);

      for await (const action of generator) {
        this.logger.info(
          `Worker ${this.name} received action ${action.actionId}:${action.actionType}`
        );

        void this.handleAction(action);
      }
    } catch (e: any) {
      this.setStatus(workerStatus.UNHEALTHY);
      if (this.killing) {
        this.logger.info(`Exiting worker, ignoring error: ${e.message}`);
        return;
      }
      this.logger.error(`Could not run worker: ${e.message}`);
      throw new HatchetError(`Could not run worker: ${e.message}`);
    }
  }

  async handleAction(action: Action) {
    const type = action.actionType
      ? actionTypeFromJSON(action.actionType)
      : ActionType.START_STEP_RUN;
    if (type === ActionType.START_STEP_RUN) {
      await this.handleStartStepRun(action);
    } else if (type === ActionType.CANCEL_STEP_RUN) {
      await this.handleCancelStepRun(action);
    } else if (type === ActionType.START_GET_GROUP_KEY) {
      await this.handleStartGroupKeyRun(action);
    } else if (type === ActionType.START_BATCH) {
      this.handleStartBatchAction(action);
    } else {
      this.logger.error(`Worker ${this.name} received unknown action type ${type}`);
    }
  }

  async upsertLabels(labels: WorkerLabels) {
    this.labels = labels;

    if (!this.workerId) {
      this.logger.warn('Worker not registered.');
      return this.labels;
    }

    this.client._v0.dispatcher.upsertWorkerLabels(this.workerId, labels);

    return this.labels;
  }
}

function toPbWorkerLabel(
  in_: CreateStep<unknown, unknown>['worker_labels']
): Record<string, DesiredWorkerLabels> {
  if (!in_) {
    return {};
  }

  return Object.entries(in_).reduce<Record<string, DesiredWorkerLabels>>(
    (acc, [key, value]) => {
      if (!value) {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: undefined,
          },
        };
      }

      if (typeof value === 'string') {
        return {
          ...acc,
          [key]: {
            strValue: value,
            intValue: undefined,
          },
        };
      }

      if (typeof value === 'number') {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: value,
          },
        };
      }

      return {
        ...acc,
        [key]: {
          strValue: typeof value.value === 'string' ? value.value : undefined,
          intValue: typeof value.value === 'number' ? value.value : undefined,
          required: value.required,
          weight: value.weight,
          comparator: value.comparator,
        },
      };
    },
    {} as Record<string, DesiredWorkerLabels>
  );
}

function onFailureTaskName(workflow: WorkflowDefinition) {
  return `${workflow.name}:on-failure-task`;
}

type LeafableTask = CreateWorkflowTaskOpts<any, any> | CreateWorkflowDurableTaskOpts<any, any>;

function getLeaves(tasks: LeafableTask[]): LeafableTask[] {
  return tasks.filter((task) => isLeafTask(task, tasks));
}

function isLeafTask(task: LeafableTask, allTasks: LeafableTask[]): boolean {
  return !allTasks.some((t) => t.parents?.some((p) => p.name === task.name));
}
