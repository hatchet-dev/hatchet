/* eslint-disable no-underscore-dangle */
/* eslint-disable no-nested-ternary */
import { LegacyHatchetClient } from '@clients/hatchet-client';
import HatchetError from '@util/errors/hatchet-error';
import {
  Action,
  ActionKey,
  ActionListener,
  createActionKey,
} from '@clients/dispatcher/action-listener';
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
import { WebhookHandler } from '@clients/worker/handler';
import { WebhookWorkerCreateRequest } from '@clients/rest/generated/data-contracts';
import { WorkflowDefinition } from '@hatchet/v1';
import { CreateWorkflowTaskOpts, NonRetryableError } from '@hatchet/v1/task';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { V0Context, CreateStep, V0DurableContext, mapRateLimit, StepRunFunction } from '../../step';
import { WorkerLabels } from '../dispatcher/dispatcher-client';

export type ActionRegistry = Record<Action['actionId'], Function>;

export interface WorkerOpts {
  name: string;
  handleKill?: boolean;
  maxRuns?: number;
  labels?: WorkerLabels;
}

export class V0Worker {
  client: LegacyHatchetClient;
  name: string;
  workerId: string | undefined;
  killing: boolean;
  handle_kill: boolean;

  action_registry: ActionRegistry;
  workflow_registry: Array<WorkflowDefinition | Workflow> = [];
  listener: ActionListener | undefined;
  futures: Record<ActionKey, HatchetPromise<any>> = {};
  contexts: Record<ActionKey, V0Context<any, any>> = {};
  maxRuns?: number;

  logger: Logger;

  registeredWorkflowPromises: Array<Promise<any>> = [];

  labels: WorkerLabels = {};

  constructor(
    client: LegacyHatchetClient,
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

    process.on('SIGTERM', () => this.exitGracefully(true));
    process.on('SIGINT', () => this.exitGracefully(true));

    this.killing = false;
    this.handle_kill = options.handleKill === undefined ? true : options.handleKill;

    this.logger = client.config.logger(`Worker/${this.name}`, this.client.config.log_level);
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
    // TODO v1
    for (const workflow of workflows) {
      const wf: Workflow = {
        ...workflow,
        id: applyNamespace(workflow.id, this.client.config.namespace),
      };

      this.registerActions(wf);
    }

    return new WebhookHandler(this, workflows);
  }

  async registerWebhook(webhook: WebhookWorkerCreateRequest) {
    return this.client.admin.registerWebhook({ ...webhook });
  }

  /**
   * @deprecated use registerWorkflow instead
   */
  async register_workflow(initWorkflow: Workflow) {
    return this.registerWorkflow(initWorkflow);
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

      const registeredWorkflow = this.client.admin.putWorkflow({
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
      const context = new V0DurableContext(action, this.client, this);
      this.contexts[createActionKey(action)] = context;

      const step = this.action_registry[actionId];

      if (!step) {
        this.logger.error(`Registered actions: '${Object.keys(this.action_registry).join(', ')}'`);
        this.logger.error(`Could not find step '${actionId}'`);
        return;
      }

      const run = async () => {
        const { middleware } = this.client.config;

        if (middleware?.before) {
          const hooks = Array.isArray(middleware.before) ? middleware.before : [middleware.before];
          for (const hook of hooks) {
            const extra = await hook(context.input, context as any);
            if (extra !== undefined) {
              const merged = { ...(context.input as any), ...extra };
              (context as any).input = merged;
              if ((context as any).data && typeof (context as any).data === 'object') {
                (context as any).data.input = merged;
              }
            }
          }
        }

        let result: any = await step(context);

        if (middleware?.after) {
          const hooks = Array.isArray(middleware.after) ? middleware.after : [middleware.after];
          for (const hook of hooks) {
            const extra = await hook(result, context as any, context.input);
            if (extra !== undefined) {
              result = { ...result, ...extra };
            }
          }
        }

        return result;
      };

      const success = async (result: any) => {
        this.logger.info(`Task run ${action.taskRunExternalId} succeeded`);

        try {
          // Send the action event to the dispatcher
          const event = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
            false,
            result || null,
            action.retryCount
          );
          await this.client.dispatcher.sendStepActionEvent(event);
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
            await this.client.dispatcher.sendStepActionEvent(failureEvent);
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
          delete this.futures[createActionKey(action)];
          delete this.contexts[createActionKey(action)];
        }
      };

      const failure = async (error: any) => {
        this.logger.error(`Task run ${action.taskRunExternalId} failed: ${error.message}`);

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
          await this.client.dispatcher.sendStepActionEvent(event);
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        } finally {
          // delete the run from the futures
          delete this.futures[createActionKey(action)];
          delete this.contexts[createActionKey(action)];
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
      this.futures[createActionKey(action)] = future;

      // Send the action event to the dispatcher
      const event = this.getStepActionEvent(
        action,
        StepActionEventType.STEP_EVENT_TYPE_STARTED,
        false,
        undefined,
        action.retryCount
      );
      this.client.dispatcher.sendStepActionEvent(event).catch((e) => {
        this.logger.error(`Could not send action event: ${e.message}`);
      });

      try {
        await future.promise;
      } catch (e: any) {
        const message = e?.message || String(e);
        if (message.includes('Cancelled')) {
          this.logger.debug(`Task run ${action.taskRunExternalId} was cancelled`);
        } else {
          this.logger.error(
            `Could not wait for task run ${action.taskRunExternalId} to finish. ` +
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
    const { actionId } = action;

    try {
      const context = new V0Context(action, this.client, this);

      const key = createActionKey(action);

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
        this.logger.info(`Task run ${action.taskRunExternalId} succeeded`);

        try {
          // Send the action event to the dispatcher
          const event = this.getGroupKeyActionEvent(
            action,
            GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_COMPLETED,
            result
          );
          this.client.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
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
        this.logger.error(`Task run ${key} failed: ${error.message}`);

        try {
          // Send the action event to the dispatcher
          const event = this.getGroupKeyActionEvent(
            action,
            GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_FAILED,
            error
          );
          this.client.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
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
      this.client.dispatcher.sendGroupKeyActionEvent(event).catch((e) => {
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
    const { taskRunExternalId } = action;
    try {
      this.logger.info(`Cancelling task run ${action.taskRunExternalId}`);
      const future = this.futures[createActionKey(action)];
      const context = this.contexts[createActionKey(action)];

      if (context && context.controller) {
        context.controller.abort('Cancelled by worker');
      }

      if (future) {
        future.promise.catch(() => {
          this.logger.info(`Cancelled task run ${action.taskRunExternalId}`);
        });
        future.cancel('Cancelled by worker');
        await future.promise;
      }
    } catch (e: any) {
      // Expected: the promise rejects when cancelled
      this.logger.debug(`Task run ${taskRunExternalId} cancellation completed`);
    } finally {
      delete this.futures[createActionKey(action)];
      delete this.contexts[createActionKey(action)];
    }
  }

  async stop() {
    await this.exitGracefully(false);
  }

  async exitGracefully(handleKill: boolean) {
    this.killing = true;

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

    if (handleKill) {
      this.logger.info('Exiting hatchet worker...');
      process.exit(0);
    }
  }

  async start() {
    // ensure all workflows are registered
    await Promise.all(this.registeredWorkflowPromises);

    if (Object.keys(this.action_registry).length === 0) {
      return;
    }

    try {
      this.listener = await this.client.dispatcher.getActionListener({
        workerName: this.name,
        services: ['default'],
        actions: Object.keys(this.action_registry),
        maxRuns: this.maxRuns,
        labels: this.labels,
      });

      this.workerId = this.listener.workerId;

      const generator = this.listener.actions();

      this.logger.info(`Worker ${this.name} listening for actions`);

      for await (const action of generator) {
        this.logger.info(
          `Worker ${this.name} received action ${action.actionId}:${action.actionType}`
        );

        void this.handleAction(action);
      }
    } catch (e: any) {
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

    this.client.dispatcher.upsertWorkerLabels(this.workerId, labels);

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

function getLeaves(tasks: CreateWorkflowTaskOpts<any, any>[]): CreateWorkflowTaskOpts<any, any>[] {
  return tasks.filter((task) => isLeafTask(task, tasks));
}

function isLeafTask(
  task: CreateWorkflowTaskOpts<any, any>,
  allTasks: CreateWorkflowTaskOpts<any, any>[]
): boolean {
  return !allTasks.some((t) => t.parents?.some((p) => p.name === task.name));
}
