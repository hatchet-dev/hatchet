import { HatchetClient } from '@clients/hatchet-client';
import HatchetError from '@util/errors/hatchet-error';
import { Action, ActionListener } from '@clients/dispatcher/action-listener';
import { ActionEvent, ActionEventType, ActionType } from '@protoc/dispatcher';
import HatchetPromise from '@util/hatchet-promise/hatchet-promise';
import { Context } from '../../step';

export class Worker {
  client: HatchetClient;
  name: string;
  killing: boolean;
  handle_kill: boolean;

  action_registry: Record<Action['actionId'], Function>;

  listener: ActionListener | undefined;

  futures: Record<Action['stepRunId'], HatchetPromise<any>> = {};

  constructor(client: HatchetClient, options: { name: string; handleKill?: boolean }) {
    this.client = client;
    this.name = options.name;
    this.action_registry = {};

    process.on('SIGTERM', () => this.exit_gracefully());
    process.on('SIGINT', () => this.exit_gracefully());

    this.killing = false;
    this.handle_kill = options.handleKill === undefined ? true : options.handleKill;
  }

  handle_start_step_run(action: Action) {
    const { actionId } = action;
    const context = new Context(action.actionPayload);

    const step = this.action_registry[actionId];
    if (!step) {
      // TODO logger.error(`Could not find step '${actionId}'`);
      return;
    }

    const run = async () => {
      return step(context.workflow_input(), context);
    };

    const success = (result: any) => {
      // TODO logger.info(`Step run ${action.stepRunId} succeeded`)

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
        // TODO logger.error(`Could not send action event: ${e.message}`)
      }
    };

    const failure = (error: any) => {
      // TODO logger.error(`Step run ${action.stepRunId} failed: ${error.message}`)

      try {
        // Send the action event to the dispatcher
        const event = this.get_action_event(action, ActionEventType.STEP_EVENT_TYPE_FAILED, error);
        this.client.dispatcher.send_action_event(event);
        // delete the run from the futures
        delete this.futures[action.stepRunId];
      } catch (e: any) {
        // TODO logger.error(`Could not send action event: ${e.message}`)
      }
    };

    const future = new HatchetPromise(run().then(success).catch(failure));
    this.futures[action.stepRunId] = future;

    try {
      // Send the action event to the dispatcher
      const event = this.get_action_event(action, ActionEventType.STEP_EVENT_TYPE_STARTED);
      this.client.dispatcher.send_action_event(event);
    } catch (e: any) {
      // TODO logger.error(`Could not send action event: ${e.message}`)
    }
  }

  get_action_event(action: Action, eventType: ActionEventType, payload: any = ''): ActionEvent {
    return {
      tenantId: action.tenantId,
      workerId: this.name,
      jobId: action.jobId,
      jobRunId: action.jobRunId,
      stepId: action.stepId,
      stepRunId: action.stepRunId,
      actionId: action.actionId,
      eventTimestamp: new Date(),
      eventType,
      eventPayload: JSON.stringify(payload), // TODO verify we need this
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

  exit_gracefully() {
    this.killing = true;

    // TODO logger.info("Gracefully exiting hatchet worker...")

    try {
      this.listener?.unregister();
    } catch (e: any) {
      // TODO logger.error(`Could not unregister listener: ${e.message}`);
    }

    // TODO wait for futures to finish

    if (this.handle_kill) {
      // TODO logger.info("Exiting hatchet worker...")
      process.exit(0);
    }
  }

  async start(retryCount = 1) {
    try {
      this.listener = await this.client.dispatcher.get_action_listener({
        workerName: this.name,
        services: ['default'],
        actions: Object.keys(this.action_registry),
      });

      const generator = this.listener.actions();

      for await (const action of generator) {
        if (action.actionType === ActionType.START_STEP_RUN) {
          this.handle_start_step_run(action);
        } else if (action.actionType === ActionType.CANCEL_STEP_RUN) {
          this.handle_cancel_step_run(action);
        }
      }
    } catch (e: any) {
      // TODO logger.error(`Could not start worker: ${e.message}`);
    }

    if (this.killing) return;

    if (retryCount > 5) {
      throw new HatchetError('Could not start worker after 5 retries');
    }

    // await this.start(retryCount + 1);
    console.log(`Could not start worker, retrying in ${retryCount} seconds`);
    // TODO retry not implemented
  }
}
