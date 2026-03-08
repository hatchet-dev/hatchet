import { DispatcherClient as PbDispatcherClient, AssignedAction } from '@hatchet/protoc/dispatcher';

import { Status } from 'nice-grpc';
import { getGrpcErrorCode } from '@util/grpc-error';
import { isAbortError } from 'abort-controller-x';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import sleep from '@util/sleep';
import HatchetError, { getErrorMessage, toHatchetError } from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';

import { DispatcherClient } from './dispatcher-client';
import { Heartbeat } from './heartbeat/heartbeat-controller';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5000; // milliseconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 20;

enum ListenStrategy {
  LISTEN_STRATEGY_V1 = 1,
  LISTEN_STRATEGY_V2 = 2,
}

export type ActionKey = `${string}/${number}`;

export type Action = AssignedAction & { readonly key: ActionKey };

export function createAction(assignedAction: AssignedAction): Action {
  const action = assignedAction as Action;
  Object.defineProperty(action, 'key', {
    get(): ActionKey {
      return `${this.taskRunExternalId}/${this.retryCount}`;
    },
    enumerable: true,
    configurable: true,
  });
  return action;
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
  done = false;
  listenStrategy = ListenStrategy.LISTEN_STRATEGY_V2;
  heartbeat: Heartbeat;
  abortController?: AbortController;

  constructor(
    client: DispatcherClient,
    workerId: string,
    retryInterval: number = DEFAULT_ACTION_LISTENER_RETRY_INTERVAL,
    retryCount: number = DEFAULT_ACTION_LISTENER_RETRY_COUNT
  ) {
    this.config = client.config;
    this.client = client.client;
    this.workerId = workerId;
    this.logger = client.config.logger(`ActionListener`, this.config.log_level);
    this.retryInterval = retryInterval;
    this.retryCount = retryCount;
    this.heartbeat = new Heartbeat(client, workerId);
  }

  actions = () =>
    (async function* gen(client: ActionListener) {
      while (true) {
        if (client.done) {
          break;
        }

        try {
          const listenClient = await client.getListenClient();

          for await (const assignedAction of listenClient) {
            yield createAction(assignedAction);
          }
        } catch (e: unknown) {
          // If the stream was aborted (e.g., during worker shutdown), exit gracefully
          if (isAbortError(e)) {
            client.logger.info('Listener aborted, exiting generator');
            break;
          }

          client.logger.info('Listener error');

          // if this is a HatchetError, we should throw this error
          if (e instanceof HatchetError) {
            throw e;
          }

          if (
            (await client.getListenStrategy()) === ListenStrategy.LISTEN_STRATEGY_V2 &&
            getGrpcErrorCode(e) === Status.UNIMPLEMENTED
          ) {
            client.setListenStrategy(ListenStrategy.LISTEN_STRATEGY_V1);
          }

          client.incrementRetries();
          client.logger.error(`Listener encountered an error: ${getErrorMessage(e)}`);
          if (client.retries > 1) {
            client.logger.info(`Retrying in ${client.retryInterval}ms...`);
            await sleep(client.retryInterval);
          } else {
            client.logger.info(`Retrying`);
          }
        }
      }
    })(this);
  async setListenStrategy(strategy: ListenStrategy) {
    this.listenStrategy = strategy;
  }

  async getListenStrategy(): Promise<ListenStrategy> {
    return this.listenStrategy;
  }

  async incrementRetries() {
    this.retries += 1;
  }

  async getListenClient(): Promise<AsyncIterable<AssignedAction>> {
    const currentTime = Math.floor(Date.now());

    // attempt to account for the time it takes to establish the listener
    if (currentTime - this.lastConnectionAttempt > this.retryInterval * 4) {
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
      // Create a new AbortController for this connection
      this.abortController = new AbortController();

      if (this.listenStrategy === ListenStrategy.LISTEN_STRATEGY_V1) {
        const result = this.client.listen(
          {
            workerId: this.workerId,
          },
          {
            signal: this.abortController.signal,
          }
        );
        this.logger.green('Connection established using LISTEN_STRATEGY_V1');
        return result;
      }

      const res = this.client.listenV2(
        {
          workerId: this.workerId,
        },
        {
          signal: this.abortController.signal,
        }
      );

      await this.heartbeat.start();
      this.logger.green('Connection established using LISTEN_STRATEGY_V2');
      return res;
    } catch (e: unknown) {
      this.retries += 1;
      this.logger.error(`Attempt ${this.retries}: Failed to connect, retrying...`);

      if (getGrpcErrorCode(e) === Status.UNAVAILABLE) {
        // Connection lost, reset heartbeat interval and retry connection
        this.heartbeat.stop();
        return this.getListenClient();
      }

      throw e;
    }
  }

  async unregister() {
    this.done = true;
    this.heartbeat.stop();

    // Abort the gRPC stream to immediately cancel the generator
    if (this.abortController) {
      this.abortController.abort('Worker stopping');
    }

    try {
      return await this.client.unsubscribe({
        workerId: this.workerId,
      });
    } catch (e: unknown) {
      throw toHatchetError(e, {
        defaultMessage: 'Failed to unsubscribe',
        prefix: 'Failed to unsubscribe: ',
      });
    }
  }
}
