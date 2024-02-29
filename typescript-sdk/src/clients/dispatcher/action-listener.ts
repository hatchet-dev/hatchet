import { DispatcherClient as PbDispatcherClient, AssignedAction } from '@hatchet/protoc/dispatcher';

import { Status } from 'nice-grpc';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import sleep from '@util/sleep';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';

import { DispatcherClient } from './dispatcher-client';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5000; // milliseconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5;

export interface Action {
  tenantId: string;
  jobId: string;
  jobName: string;
  jobRunId: string;
  stepId: string;
  stepRunId: string;
  actionId: string;
  actionType: number;
  actionPayload: string;
  workflowRunId: string;
  getGroupKeyRunId: string;
  stepName: string;
}

export class ActionListener {
  config: ClientConfig;
  client: PbDispatcherClient;
  workerId: string;
  logger: Logger;
  lastConnectionAttempt: number = 0;
  retries: number = 0;
  retryInterval: number = DEFAULT_ACTION_LISTENER_RETRY_INTERVAL;
  retryCount: number = DEFAULT_ACTION_LISTENER_RETRY_COUNT;

  constructor(
    client: DispatcherClient,
    workerId: string,
    retryInterval: number = DEFAULT_ACTION_LISTENER_RETRY_INTERVAL,
    retryCount: number = DEFAULT_ACTION_LISTENER_RETRY_COUNT
  ) {
    this.config = client.config;
    this.client = client.client;
    this.workerId = workerId;
    this.logger = new Logger(`ActionListener`, this.config.log_level);
    this.retryInterval = retryInterval;
    this.retryCount = retryCount;
  }

  actions = () =>
    (async function* gen(client: ActionListener) {
      while (true) {
        try {
          for await (const assignedAction of await client.getListenClient()) {
            const action: Action = {
              ...assignedAction,
            };

            yield action;
          }
        } catch (e: any) {
          // if this is a HatchetError, we should throw this error
          if (e instanceof HatchetError) {
            throw e;
          }

          if (e.code === Status.CANCELLED) {
            break;
          }

          client.incrementRetries();
        }
      }
    })(this);

  async incrementRetries() {
    this.retries += 1;
  }

  async getListenClient(): Promise<AsyncIterable<AssignedAction>> {
    const currentTime = Math.floor(Date.now());

    // subtract 1000 from the last connection attempt to account for the time it takes to establish the listener
    if (currentTime - this.lastConnectionAttempt - 1000 > this.retryInterval) {
      this.retries = 0;
    }

    this.lastConnectionAttempt = currentTime;

    if (this.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT) {
      throw new HatchetError(
        `Could not subscribe to the worker after ${DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries`
      );
    }

    this.logger.info(
      `Connecting to Hatchet to establish listener for actions... ${this.retries}/${DEFAULT_ACTION_LISTENER_RETRY_COUNT} (last attempt: ${this.lastConnectionAttempt})`
    );

    if (this.retries >= 1) {
      await sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL);
    }

    try {
      return this.client.listen({
        workerId: this.workerId,
      });
    } catch (e: any) {
      this.retries += 1;
      this.logger.error(`Attempt ${this.retries}: Failed to connect, retrying...`); // Optional: log retry attempt

      return this.getListenClient();
    }
  }

  async unregister() {
    try {
      return this.client.unsubscribe({
        workerId: this.workerId,
      });
    } catch (e: any) {
      throw new HatchetError(`Failed to unsubscribe: ${e.message}`);
    }
  }
}
