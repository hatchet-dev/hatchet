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
import {
  CreateStepRateLimit,
  DesiredWorkerLabels,
  StickyStrategy,
} from '@hatchet/protoc/workflows';
import { actionMap, Logger, taskRunLog } from '@hatchet/util/logger';
import { BaseWorkflowDeclaration, WorkflowDefinition, HatchetClient } from '@hatchet/v1';
import { CreateTaskOpts } from '@hatchet/protoc/v1/workflows';
import {
  CreateOnFailureTaskOpts,
  CreateOnSuccessTaskOpts,
  CreateWorkflowDurableTaskOpts,
  CreateWorkflowTaskOpts,
  NonRetryableError,
} from '@hatchet/v1/task';
import { taskConditionsToPb } from '@hatchet/v1/conditions/transformer';
import { zodToJsonSchema } from 'zod-to-json-schema';

import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import sleep from '@hatchet/util/sleep';
import { throwIfAborted } from '@hatchet/util/abort-error';
import { Context, DurableContext } from './context';
import { parentRunContextManager } from '../../parent-run-context-vars';
import { HealthServer, workerStatus, type WorkerStatus } from './health-server';
import { SlotConfig, SlotType } from '../../slot-types';

export type ActionRegistry = Record<Action['actionId'], Function>;

export interface WorkerOpts {
  name: string;
  handleKill?: boolean;
  slots?: number;
  durableSlots?: number;
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
  workflow_registry: Array<WorkflowDefinition> = [];
  listener: ActionListener | undefined;
  futures: Record<Action['taskRunExternalId'], HatchetPromise<any>> = {};
  contexts: Record<Action['taskRunExternalId'], Context<any, any>> = {};
  slots?: number;
  durableSlots?: number;
  slotConfig: SlotConfig;

  logger: Logger;

  registeredWorkflowPromises: Array<Promise<any>> = [];

  labels: WorkerLabels = {};

  healthPort: number;
  enableHealthServer: boolean;

  private healthServer: HealthServer | undefined;
  private status: WorkerStatus = workerStatus.INITIALIZED;

  constructor(
    client: HatchetClient,
    options: {
      name: string;
      handleKill?: boolean;
      slots?: number;
      durableSlots?: number;
      slotConfig?: SlotConfig;
      labels?: WorkerLabels;
    }
  ) {
    this.client = client;
    this.name = applyNamespace(options.name, this.client.config.namespace);
    this.action_registry = {};
    this.slots = options.slots;
    this.durableSlots = options.durableSlots;
    this.slotConfig = options.slotConfig || {};

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

  // TODO where is this used, this doesnt make much sense
  private getAvailableSlots(): number {
    const baseSlots = this.slotConfig[SlotType.Default] ?? this.slots ?? 0;
    if (!baseSlots) {
      return 0;
    }
    const currentRuns = Object.keys(this.futures).length;
    return Math.max(0, baseSlots - currentRuns);
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

  private registerActionsV1(workflow: WorkflowDefinition) {
    const newActions = workflow._tasks
      .filter((task) => !!task.fn)
      .reduce<ActionRegistry>((acc, task) => {
        acc[`${workflow.name}:${task.name.toLowerCase()}`] = (ctx: Context<any, any>) =>
          task.fn!(ctx.input, ctx);
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
          isDurable: false,
          slotRequests: { default: 1 },
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
          rateLimits: mapRateLimitPb(onFailure.rateLimits || workflow.taskDefaults?.rateLimits),
          workerLabels: mapWorkerLabelPb(
            onFailure.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
          ),
          concurrency: [],
          backoffFactor: onFailure.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
          backoffMaxSeconds:
            onFailure.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
          isDurable: false,
          slotRequests: { default: 1 },
        };
      }

      let onSuccessTask: CreateWorkflowTaskOpts<any, any> | undefined;

      if (!durable && workflow.onSuccess && typeof workflow.onSuccess === 'function') {
        const parents = getLeaves([...workflow._tasks, ...workflow._durableTasks]);

        onSuccessTask = {
          name: 'on-success-task',
          fn: workflow.onSuccess,
          executionTimeout: '60s',
          parents,
          retries: 0,
          rateLimits: [],
          desiredWorkerLabels: undefined,
          concurrency: [],
        };
      }

      if (!durable && workflow.onSuccess && typeof workflow.onSuccess === 'object') {
        const onSuccess = workflow.onSuccess as CreateOnSuccessTaskOpts<any, any>;
        const parents = getLeaves([...workflow._tasks, ...workflow._durableTasks]);

        onSuccessTask = {
          name: 'on-success-task',
          fn: onSuccess.fn,
          executionTimeout:
            onSuccess.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s',
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
        ...(workflow.on && 'event' in workflow.on && workflow.on.event
          ? Array.isArray(workflow.on.event)
            ? workflow.on.event.map((event) => applyNamespace(event, this.client.config.namespace))
            : [applyNamespace(workflow.on.event, this.client.config.namespace)]
          : []),
      ];
      const cronTriggers: string[] = [
        ...(workflow.onCrons || []),
        ...(workflow.on && 'cron' in workflow.on && workflow.on.cron
          ? Array.isArray(workflow.on.cron)
            ? workflow.on.cron
            : [workflow.on.cron]
          : []),
      ];

      const concurrencyArr = Array.isArray(concurrency) ? concurrency : [];
      const concurrencySolo = !Array.isArray(concurrency) ? concurrency : undefined;

      // Convert Zod schema to JSON Schema if provided
      let inputJsonSchema: Uint8Array | undefined;
      if (workflow.inputValidator) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const jsonSchema = zodToJsonSchema(workflow.inputValidator as any);
        inputJsonSchema = new TextEncoder().encode(JSON.stringify(jsonSchema));
      }

      const durableTaskSet = new Set(workflow._durableTasks);

      let stickyStrategy: StickyStrategy | undefined;
      // `workflow.sticky` is a v1 (non-protobuf) config which may also include legacy protobuf
      // enum values for backwards compatibility.
      switch (workflow.sticky) {
        case 'soft':
        case 'SOFT':
        case 0:
          stickyStrategy = StickyStrategy.SOFT;
          break;
        case 'hard':
        case 'HARD':
        case 1:
          stickyStrategy = StickyStrategy.HARD;
          break;
        default:
          throw new HatchetError(`Invalid sticky strategy: ${workflow.sticky}`);
      }

      const registeredWorkflow = this.client._v0.admin.putWorkflowV1({
        name: workflow.name,
        description: workflow.description || '',
        version: workflow.version || '',
        eventTriggers,
        cronTriggers,
        sticky: stickyStrategy,
        concurrencyArr,
        onFailureTask,
        defaultPriority: workflow.defaultPriority,
        inputJsonSchema,
        tasks: [...workflow._tasks, ...workflow._durableTasks].map<CreateTaskOpts>((task) => ({
          readableId: task.name,
          action: `${workflow.name}:${task.name}`,
          timeout: task.executionTimeout || workflow.taskDefaults?.executionTimeout || '60s',
          scheduleTimeout: task.scheduleTimeout || workflow.taskDefaults?.scheduleTimeout,
          inputs: '{}',
          parents: task.parents?.map((p) => p.name) ?? [],
          userData: '{}',
          retries: task.retries || workflow.taskDefaults?.retries || 0,
          rateLimits: mapRateLimitPb(task.rateLimits || workflow.taskDefaults?.rateLimits),
          workerLabels: mapWorkerLabelPb(
            task.desiredWorkerLabels || workflow.taskDefaults?.workerLabels
          ),
          backoffFactor: task.backoff?.factor || workflow.taskDefaults?.backoff?.factor,
          backoffMaxSeconds: task.backoff?.maxSeconds || workflow.taskDefaults?.backoff?.maxSeconds,
          conditions: taskConditionsToPb(task),
          isDurable: durableTaskSet.has(task),
          slotRequests:
            task.slotRequests || (durableTaskSet.has(task) ? { durable: 1 } : { default: 1 }),
          concurrency: task.concurrency
            ? Array.isArray(task.concurrency)
              ? task.concurrency
              : [task.concurrency]
            : workflow.taskDefaults?.concurrency
              ? Array.isArray(workflow.taskDefaults.concurrency)
                ? workflow.taskDefaults.concurrency
                : [workflow.taskDefaults.concurrency]
              : [],
        })),
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

  async handleStartStepRun(action: Action) {
    const { actionId, taskRunExternalId, taskName } = action;

    try {
      // Note: we always use a DurableContext since its a superset of the Context class
      const context = new DurableContext(action, this.client, this);
      this.contexts[taskRunExternalId] = context;

      const step = this.action_registry[actionId];

      if (!step) {
        this.logger.error(`Registered actions: '${Object.keys(this.action_registry).join(', ')}'`);
        this.logger.error(`Could not find step '${actionId}'`);
        return;
      }

      const run = async () => {
        return parentRunContextManager.runWithContext(
          {
            parentId: action.workflowRunId,
            parentTaskRunExternalId: taskRunExternalId,
            childIndex: 0,
            desiredWorkerId: this.workerId || '',
            signal: context.abortController.signal,
          },
          () => {
            // Precheck: if cancellation already happened, don't execute user code.
            throwIfAborted(context.abortController.signal);
            return step(context);
          }
        );
      };

      const success = async (result: any) => {
        try {
          if (context.cancelled) {
            return;
          }

          this.logger.info(taskRunLog(taskName, taskRunExternalId, 'completed'));

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
          delete this.futures[taskRunExternalId];
          delete this.contexts[taskRunExternalId];
        }
      };

      const failure = async (error: any) => {
        const shouldNotRetry = error instanceof NonRetryableError;

        try {
          if (context.cancelled) {
            return;
          }

          this.logger.error(taskRunLog(taskName, taskRunExternalId, `failed: ${error.message}`));

          if (error.stack) {
            this.logger.error(error.stack);
          }

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
          delete this.futures[taskRunExternalId];
          delete this.contexts[taskRunExternalId];
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

          // Postcheck: user code may swallow AbortError; don't report completion after cancellation.
          // If we reached this point and the signal is aborted, the task likely caught/ignored cancellation.
          if (context.abortController.signal.aborted) {
            this.logger.warn(
              `Cancellation: task run ${taskRunExternalId} returned after cancellation was signaled. ` +
                `This usually means an AbortError was caught and not propagated. ` +
                `See https://docs.hatchet.run/home/cancellation`
            );
            return;
          }
          throwIfAborted(context.abortController.signal);

          await success(result);
        })()
      );
      this.futures[taskRunExternalId] = future;

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
        const message = e?.message || String(e);
        // TODO is this cased correctly...
        if (!message.includes('Cancelled')) {
          this.logger.error(
            `Could not wait for task run ${taskRunExternalId} to finish. ` +
              `See https://docs.hatchet.run/home/cancellation for best practices on handling cancellation: `,
            e
          );
        }
      }
    } catch (e: any) {
      this.logger.error('Could not send action event (outer): ', e);
    }
  }

  async handleStartGroupKeyRun(action: Action) {
    const { actionId, getGroupKeyRunId, taskRunExternalId, taskName } = action;

    this.logger.error(
      'Concurrency Key Functions have been deprecated and will be removed in a future release. Use Concurrency Expressions instead.'
    );

    try {
      const context = new Context(action, this.client, this);

      const key = getGroupKeyRunId;

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
        this.logger.info(taskRunLog(taskName, taskRunExternalId, 'completed'));

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
        this.logger.error(taskRunLog(taskName, taskRunExternalId, `failed: ${error.message}`));

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
      taskId: action.taskId,
      taskRunExternalId: action.taskRunExternalId,
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
    const { taskRunExternalId, taskName } = action;

    try {
      const future = this.futures[taskRunExternalId];
      const context = this.contexts[taskRunExternalId];

      if (context && context.abortController) {
        context.abortController.abort('Cancelled by worker'); // TODO this reason is nonsensical
      }

      if (future) {
        const start = Date.now();
        const warningThresholdMs = this.client.config.cancellation_warning_threshold ?? 300;
        const gracePeriodMs = this.client.config.cancellation_grace_period ?? 1000;
        const warningMs = Math.max(0, warningThresholdMs);
        const graceMs = Math.max(0, gracePeriodMs);

        // Ensure cancelling this future doesn't create an unhandled rejection in cases
        // where the main action handler isn't currently awaiting `future.promise`.
        future.promise.catch(() => undefined);

        // Cancel the future (rejects the wrapper); user code must still cooperate with AbortSignal.
        future.cancel('Cancelled by worker'); // TODO this reason is nonsensical

        // Track completion of the underlying work (not the cancelable wrapper).
        // Ensure this promise never throws into our supervision flow.
        const completion = (future.inner ?? future.promise).catch(() => undefined);

        // Wait until warning threshold, then log if still running.
        if (warningMs > 0) {
          const winner = await Promise.race([
            completion.then(() => 'done' as const),
            sleep(warningMs).then(() => 'warn' as const),
          ]);

          if (winner === 'warn') {
            const milliseconds = Date.now() - start;
            this.logger.warn(
              `Cancellation: task run ${taskRunExternalId} has not cancelled after ${milliseconds}ms. Consider checking for blocking operations. ` +
                `See https://docs.hatchet.run/home/cancellation`
            );
          }
        }

        // Wait until grace period (total), then log if still running.
        const elapsedMs = Date.now() - start;
        const remainingMs = graceMs - elapsedMs;
        const winner = await Promise.race([
          completion.then(() => 'done' as const),
          sleep(Math.max(0, remainingMs)).then(() => 'grace' as const),
        ]);

        if (winner === 'done') {
          this.logger.info(taskRunLog(taskName, taskRunExternalId, 'cancelled'));
        } else {
          const totalElapsedMs = Date.now() - start;
          this.logger.error(
            `Cancellation: task run ${taskRunExternalId} still running after cancellation grace period ` +
              `${totalElapsedMs}ms.\n` +
              `JavaScript cannot force-kill user code; see: https://docs.hatchet.run/home/cancellation`
          );
        }
      }
    } catch (e: any) {
      this.logger.error(
        `Cancellation: error while supervising cancellation for task run ${taskRunExternalId}: ${e?.message || e}`
      );
    } finally {
      delete this.futures[taskRunExternalId];
      delete this.contexts[taskRunExternalId];
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
        slotConfig: this.slotConfig,
        labels: this.labels,
      });

      this.workerId = this.listener.workerId;
      this.setStatus(workerStatus.HEALTHY);

      const generator = this.listener.actions();

      this.logger.info(`Worker ${this.name} listening for actions`);

      for await (const action of generator) {
        const receivedType = actionMap(action.actionType);

        this.logger.info(taskRunLog(action.taskName, action.taskRunExternalId, `${receivedType}`));

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

function mapWorkerLabelPb(
  in_: CreateWorkflowTaskOpts<any, any>['desiredWorkerLabels']
): Record<string, DesiredWorkerLabels> {
  if (!in_) {
    return {};
  }

  return Object.entries(in_).reduce<Record<string, DesiredWorkerLabels>>(
    (acc, [key, label]) => {
      if (!label) {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: undefined,
          },
        };
      }

      if (typeof label === 'string') {
        return {
          ...acc,
          [key]: {
            strValue: label,
            intValue: undefined,
          },
        };
      }

      if (typeof label === 'number') {
        return {
          ...acc,
          [key]: {
            strValue: undefined,
            intValue: label,
          },
        };
      }

      return {
        ...acc,
        [key]: {
          strValue: typeof label.value === 'string' ? label.value : undefined,
          intValue: typeof label.value === 'number' ? label.value : undefined,
          required: label.required,
          weight: label.weight,
          comparator: label.comparator,
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

export function mapRateLimitPb(
  limits: CreateWorkflowTaskOpts<any, any>['rateLimits']
): CreateStepRateLimit[] {
  if (!limits) return [];

  return limits.map((l) => {
    let key = l.staticKey;
    const keyExpression = l.dynamicKey;

    if (l.key !== undefined) {
      // eslint-disable-next-line no-console
      console.warn(
        'key is deprecated and will be removed in a future release, please use staticKey instead'
      );
      key = l.key;
    }

    if (keyExpression !== undefined) {
      if (key !== undefined) {
        throw new Error('Cannot have both static key and dynamic key set');
      }
      key = keyExpression;
      if (!validateCelExpression(keyExpression)) {
        throw new Error(`Invalid CEL expression: ${keyExpression}`);
      }
    }

    if (key === undefined) {
      throw new Error(`Invalid key`);
    }

    let units: number | undefined;
    let unitsExpression: string | undefined;
    if (typeof l.units === 'number') {
      units = l.units;
    } else {
      if (!validateCelExpression(l.units)) {
        throw new Error(`Invalid CEL expression: ${l.units}`);
      }
      unitsExpression = l.units;
    }

    let limitExpression: string | undefined;
    if (l.limit !== undefined) {
      if (typeof l.limit === 'number') {
        limitExpression = `${l.limit}`;
      } else {
        if (!validateCelExpression(l.limit)) {
          throw new Error(`Invalid CEL expression: ${l.limit}`);
        }

        limitExpression = l.limit;
      }
    }

    if (keyExpression !== undefined && limitExpression === undefined) {
      throw new Error('CEL based keys requires limit to be set');
    }

    if (limitExpression === undefined) {
      limitExpression = `-1`;
    }

    return {
      key,
      keyExpr: keyExpression,
      units,
      unitsExpr: unitsExpression,
      limitValuesExpr: limitExpression,
      duration: l.duration,
    };
  });
}

// Helper function to validate CEL expressions
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function validateCelExpression(_expr: string): boolean {
  // FIXME: this is a placeholder. In a real implementation, you'd need to use a CEL parser or validator.
  // For now, we'll just return true to mimic the behavior.
  return true;
}
