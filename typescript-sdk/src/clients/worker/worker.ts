import { HatchetClient } from '@clients/hatchet-client';
import HatchetError from '@util/errors/hatchet-error';
import { Action, ActionListener } from '@clients/dispatcher/action-listener';
import {
  StepActionEvent,
  StepActionEventType,
  ActionType,
  GroupKeyActionEvent,
  GroupKeyActionEventType,
} from '@hatchet/protoc/dispatcher';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';
import { Workflow } from '@hatchet/workflow';
import {
  ConcurrencyLimitStrategy,
  CreateWorkflowStepOpts,
  WorkflowConcurrencyOpts,
} from '@hatchet/protoc/workflows';
import { Logger } from '@hatchet/util/logger';
// import sleep from '@hatchet/util/sleep';
import { Context, StepRunFunction } from '../../step';

export type ActionRegistry = Record<Action['actionId'], Function>;

export class Worker {
  client: HatchetClient;
  name: string;
  killing: boolean;
  handle_kill: boolean;

  action_registry: ActionRegistry;
  listener: ActionListener | undefined;
  futures: Record<Action['stepRunId'], HatchetPromise<any>> = {};
  contexts: Record<Action['stepRunId'], Context<any, any>> = {};
  maxRuns?: number;

  logger: Logger;

  constructor(
    client: HatchetClient,
    options: { name: string; handleKill?: boolean; maxRuns?: number }
  ) {
    this.client = client;
    this.name = options.name;
    this.action_registry = {};
    this.maxRuns = options.maxRuns;

    process.on('SIGTERM', () => this.exitGracefully());
    process.on('SIGINT', () => this.exitGracefully());

    this.killing = false;
    this.handle_kill = options.handleKill === undefined ? true : options.handleKill;

    this.logger = new Logger(`Worker/${this.name}`, this.client.config.log_level);
  }

  async registerWorkflow(workflow: Workflow) {
    try {
      const concurrency: WorkflowConcurrencyOpts | undefined = workflow.concurrency?.name
        ? {
            action: `${workflow.id}:${workflow.concurrency.name}`,
            maxRuns: workflow.concurrency.maxRuns || 1,
            limitStrategy:
              workflow.concurrency.limitStrategy || ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
          }
        : undefined;

      await this.client.admin.put_workflow({
        name: workflow.id,
        description: workflow.description,
        version: workflow.version || '',
        eventTriggers: workflow.on.event ? [workflow.on.event] : [],
        cronTriggers: workflow.on.cron ? [workflow.on.cron] : [],
        scheduledTriggers: [],
        concurrency,
        scheduleTimeout: workflow.scheduleTimeout,
        jobs: [
          {
            name: workflow.id,
            timeout: workflow.timeout || '60s',
            description: workflow.description,
            steps: workflow.steps.map<CreateWorkflowStepOpts>((step) => ({
              readableId: step.name,
              action: `${workflow.id}:${step.name}`,
              timeout: step.timeout || '60s',
              inputs: '{}',
              parents: step.parents ?? [],
              userData: '{}',
              retries: step.retries || 0,
            })),
          },
        ],
      });
    } catch (e: any) {
      throw new HatchetError(`Could not register workflow: ${e.message}`);
    }

    const newActions = workflow.steps.reduce<ActionRegistry>((acc, step) => {
      acc[`${workflow.id}:${step.name}`] = step.run;
      return acc;
    }, {});

    this.action_registry = {
      ...this.action_registry,
      ...newActions,
    };

    this.action_registry = workflow.concurrency?.name
      ? {
          ...this.action_registry,
          [`${workflow.id}:${workflow.concurrency.name}`]: workflow.concurrency.key,
        }
      : {
          ...this.action_registry,
        };
  }

  registerAction<T, K>(actionId: string, action: StepRunFunction<T, K>) {
    this.action_registry[actionId] = action;
  }

  handleStartStepRun(action: Action) {
    const { actionId } = action;

    try {
      const context = new Context(action, this.client.dispatcher, this.client.event);
      this.contexts[action.stepRunId] = context;

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
          const event = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
            result
          );
          this.client.dispatcher.sendStepActionEvent(event);

          // delete the run from the futures
          delete this.futures[action.stepRunId];
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        }
      };

      const failure = (error: any) => {
        this.logger.error(`Step run ${action.stepRunId} failed: ${error.message}`);

        try {
          // Send the action event to the dispatcher
          const event = this.getStepActionEvent(
            action,
            StepActionEventType.STEP_EVENT_TYPE_FAILED,
            error
          );
          this.client.dispatcher.sendStepActionEvent(event);
          // delete the run from the futures
          delete this.futures[action.stepRunId];
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        }
      };

      const future = new HatchetPromise(run().then(success).catch(failure));
      this.futures[action.stepRunId] = future;

      // Send the action event to the dispatcher
      const event = this.getStepActionEvent(action, StepActionEventType.STEP_EVENT_TYPE_STARTED);
      this.client.dispatcher.sendStepActionEvent(event);
    } catch (e: any) {
      this.logger.error(`Could not send action event: ${e.message}`);
    }
  }

  handleStartGroupKeyRun(action: Action) {
    const { actionId } = action;

    try {
      const context = new Context(action, this.client.dispatcher, this.client.event);

      const key = action.getGroupKeyRunId;

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
          this.client.dispatcher.sendGroupKeyActionEvent(event);

          // delete the run from the futures
          delete this.futures[key];
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
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
          this.client.dispatcher.sendGroupKeyActionEvent(event);
          // delete the run from the futures
          delete this.futures[key];
        } catch (e: any) {
          this.logger.error(`Could not send action event: ${e.message}`);
        }
      };

      const future = new HatchetPromise(run().then(success).catch(failure));
      this.futures[action.getGroupKeyRunId] = future;

      // Send the action event to the dispatcher
      const event = this.getStepActionEvent(action, StepActionEventType.STEP_EVENT_TYPE_STARTED);
      this.client.dispatcher.sendStepActionEvent(event);
    } catch (e: any) {
      this.logger.error(`Could not send action event: ${e.message}`);
    }
  }

  getStepActionEvent(
    action: Action,
    eventType: StepActionEventType,
    payload: any = ''
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
    };
  }

  getGroupKeyActionEvent(
    action: Action,
    eventType: GroupKeyActionEventType,
    payload: any = ''
  ): GroupKeyActionEvent {
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

  handleCancelStepRun(action: Action) {
    try {
      this.logger.info(`Cancelling step run ${action.stepRunId}`);

      const { stepRunId } = action;
      const future = this.futures[stepRunId];
      const context = this.contexts[stepRunId];

      if (context && context.controller) {
        context.controller.abort('Cancelled by worker');
        delete this.contexts[stepRunId];
      }

      if (future) {
        future.promise.catch(() => {
          this.logger.info(`Cancelled step run ${action.stepRunId}`);
        });
        future.cancel('Cancelled by worker');
        delete this.futures[stepRunId];
      }
    } catch (e: any) {
      this.logger.error(`Could not cancel step run: ${e.message}`);
    }
  }

  async stop() {
    await this.exitGracefully();
  }

  async exitGracefully() {
    this.killing = true;

    this.logger.info('Starting to exit...');

    try {
      this.listener?.unregister();
    } catch (e: any) {
      this.logger.error(`Could not unregister listener: ${e.message}`);
    }

    this.logger.info('Gracefully exiting hatchet worker, running tasks will attempt to finish...');

    // attempt to wait for futures to finish
    await Promise.all(Object.values(this.futures).map(({ promise }) => promise));

    if (this.handle_kill) {
      this.logger.info('Exiting hatchet worker...');
      process.exit(0);
    }
  }

  async start() {
    try {
      this.listener = await this.client.dispatcher.getActionListener({
        workerName: this.name,
        services: ['default'],
        actions: Object.keys(this.action_registry),
        maxRuns: this.maxRuns,
      });

      const generator = this.listener.actions();

      this.logger.info(`Worker ${this.name} listening for actions`);

      for await (const action of generator) {
        this.logger.info(
          `Worker ${this.name} received action ${action.actionId}:${action.actionType}`
        );

        if (action.actionType === ActionType.START_STEP_RUN) {
          this.handleStartStepRun(action);
        } else if (action.actionType === ActionType.CANCEL_STEP_RUN) {
          this.handleCancelStepRun(action);
        } else if (action.actionType === ActionType.START_GET_GROUP_KEY) {
          this.handleStartGroupKeyRun(action);
        } else {
          this.logger.error(
            `Worker ${this.name} received unknown action type ${action.actionType}`
          );
        }
      }
    } catch (e: any) {
      this.logger.error(`Could not run worker: ${e.message}`);
      throw new HatchetError(`Could not run worker: ${e.message}`);
    }
  }
}
