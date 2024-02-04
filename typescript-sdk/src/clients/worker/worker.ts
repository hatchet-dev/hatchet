import { HatchetClient } from '@clients/hatchet-client';
import HatchetError from '@util/errors/hatchet-error';
import { Action, ActionListener } from '@clients/dispatcher/action-listener';
import { ActionEvent, ActionEventType, ActionType } from '@hatchet/protoc/dispatcher';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';
import { Workflow } from '@hatchet/workflow';
import { CreateWorkflowStepOpts } from '@hatchet/protoc/workflows';
import { Logger } from '@hatchet/util/logger';
import sleep from '@hatchet/util/sleep';
import { Context } from '../../step';

export type ActionRegistry = Record<Action['actionId'], Function>;

export class Worker {
  serviceName = 'default';
  client: HatchetClient;
  name: string;
  killing: boolean;
  handle_kill: boolean;

  action_registry: ActionRegistry;
  listener: ActionListener | undefined;
  futures: Record<Action['stepRunId'], HatchetPromise<any>> = {};

  logger: Logger;

  constructor(client: HatchetClient, options: { name: string; handleKill?: boolean }) {
    this.client = client;
    this.name = options.name;
    this.action_registry = {};

    process.on('SIGTERM', () => this.exit_gracefully());
    process.on('SIGINT', () => this.exit_gracefully());

    this.killing = false;
    this.handle_kill = options.handleKill === undefined ? true : options.handleKill;

    this.logger = new Logger(`Worker/${this.name}`, this.client.config.log_level);
  }

  async register_workflow(workflow: Workflow, options?: { autoVersion?: boolean }) {
    try {
      await this.client.admin.put_workflow(
        {
          name: workflow.id,
          description: workflow.description,
          version: 'v0.55.0', // FIXME  workflow.version,
          eventTriggers: workflow.on.event ? [workflow.on.event] : [],
          cronTriggers: workflow.on.cron ? [workflow.on.cron] : [],
          scheduledTriggers: [],
          jobs: [
            {
              name: 'my-job', // FIXME variable names
              timeout: '60s',
              description: 'my-job',
              steps: workflow.steps.map<CreateWorkflowStepOpts>((step) => ({
                readableId: step.name,
                action: `${this.serviceName}:${step.name}`,
                timeout: '60s',
                inputs: '{}',
                parents: step.parents ?? [],
              })),
            },
          ],
        },
        {
          autoVersion: !options?.autoVersion,
        }
      );
    } catch (e: any) {
      throw new HatchetError(`Could not register workflow: ${e.message}`);
    }

    this.action_registry = workflow.steps.reduce<ActionRegistry>((acc, step) => {
      acc[`${this.serviceName}:${step.name}`] = step.run;
      return acc;
    }, {});
  }

  handle_start_step_run(action: Action) {
    const { actionId } = action;
    const context = new Context(action.actionPayload);

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
        const event = this.get_action_event(
          action,
          ActionEventType.STEP_EVENT_TYPE_COMPLETED,
          result
        );
        this.client.dispatcher.send_action_event(event);

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
        const event = this.get_action_event(action, ActionEventType.STEP_EVENT_TYPE_FAILED, error);
        this.client.dispatcher.send_action_event(event);
        // delete the run from the futures
        delete this.futures[action.stepRunId];
      } catch (e: any) {
        this.logger.error(`Could not send action event: ${e.message}`);
      }
    };

    const future = new HatchetPromise(run().then(success).catch(failure));
    this.futures[action.stepRunId] = future;

    try {
      // Send the action event to the dispatcher
      const event = this.get_action_event(action, ActionEventType.STEP_EVENT_TYPE_STARTED);
      this.client.dispatcher.send_action_event(event);
    } catch (e: any) {
      this.logger.error(`Could not send action event: ${e.message}`);
    }
  }

  get_action_event(action: Action, eventType: ActionEventType, payload: any = ''): ActionEvent {
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

  handle_cancel_step_run(action: Action) {
    const { stepRunId } = action;
    const future = this.futures[stepRunId];
    if (future) {
      future.cancel();
      delete this.futures[stepRunId];
    }
  }

  async stop() {
    await this.exit_gracefully();
  }

  async exit_gracefully() {
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
    let retries = 0;

    while (retries < 5) {
      try {
        this.listener = await this.client.dispatcher.get_action_listener({
          workerName: this.name,
          services: ['default'],
          actions: Object.keys(this.action_registry),
        });

        const generator = this.listener.actions();

        this.logger.info(`Worker ${this.name} listening for actions`);

        for await (const action of generator) {
          this.logger.info(`Worker ${this.name} received action ${action.actionId}`);

          if (action.actionType === ActionType.START_STEP_RUN) {
            this.handle_start_step_run(action);
          } else if (action.actionType === ActionType.CANCEL_STEP_RUN) {
            this.handle_cancel_step_run(action);
          }
        }

        break;
      } catch (e: any) {
        this.logger.error(`Could not start worker: ${e.message}`);
        retries += 1;
        const wait = 500;
        this.logger.error(`Could not start worker, retrying in ${500} seconds`);
        await sleep(wait);
      }
    }

    if (this.killing) return;

    if (retries > 5) {
      throw new HatchetError('Could not start worker after 5 retries');
    }
  }
}
