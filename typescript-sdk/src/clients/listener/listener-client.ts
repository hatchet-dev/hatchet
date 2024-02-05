import { Channel, ClientFactory, Status } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  ResourceEventType,
} from '@hatchet/protoc/dispatcher';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';
import sleep from '@hatchet/util/sleep';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5; // seconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5;

// eslint-disable-next-line no-shadow
export enum StepRunEventType {
  STEP_RUN_EVENT_TYPE_STARTED = 'STEP_RUN_EVENT_TYPE_STARTED',
  STEP_RUN_EVENT_TYPE_COMPLETED = 'STEP_RUN_EVENT_TYPE_COMPLETED',
  STEP_RUN_EVENT_TYPE_FAILED = 'STEP_RUN_EVENT_TYPE_FAILED',
  STEP_RUN_EVENT_TYPE_CANCELLED = 'STEP_RUN_EVENT_TYPE_CANCELLED',
  STEP_RUN_EVENT_TYPE_TIMED_OUT = 'STEP_RUN_EVENT_TYPE_TIMED_OUT',
}

export interface StepRunEvent {
  type: StepRunEventType;
  payload: string;
}

export class ListenerClient {
  config: ClientConfig;
  client: PbDispatcherClient;
  logger: Logger;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
    this.logger = new Logger(`Listener`, config.log_level);
  }

  async on(workflowRunId: string, handler: (event: StepRunEvent) => Promise<void>) {
    let listener = this.client.subscribeToWorkflowEvents({
      workflowRunId,
    });

    try {
      for await (const workflowEvent of listener) {
        let eventType: StepRunEventType;

        switch (workflowEvent.eventType) {
          case ResourceEventType.RESOURCE_EVENT_TYPE_STARTED:
            eventType = StepRunEventType.STEP_RUN_EVENT_TYPE_STARTED;
            break;
          case ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED:
            eventType = StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED;
            break;
          case ResourceEventType.RESOURCE_EVENT_TYPE_FAILED:
            eventType = StepRunEventType.STEP_RUN_EVENT_TYPE_FAILED;
            break;
          case ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED:
            eventType = StepRunEventType.STEP_RUN_EVENT_TYPE_CANCELLED;
            break;
          case ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT:
            eventType = StepRunEventType.STEP_RUN_EVENT_TYPE_TIMED_OUT;
            break;
          default:
            throw new HatchetError(`Unknown event type: ${workflowEvent.eventType}`);
        }

        await handler({
          type: eventType,
          payload: workflowEvent.eventPayload,
        });
      }
    } catch (e: any) {
      if (e.code === Status.CANCELLED) {
        return;
      }
      if (e.code === Status.UNAVAILABLE) {
        listener = await this.retrySubscribe(workflowRunId);
      }
    }
  }

  async retrySubscribe(workflowRunId: string) {
    let retries = 0;

    while (retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT) {
      try {
        await sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL);

        const listener = this.client.subscribeToWorkflowEvents({
          workflowRunId,
        });

        return listener;
      } catch (e: any) {
        retries += 1;
      }
    }

    throw new HatchetError(
      `Could not subscribe to the worker after ${DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries`
    );
  }
}
