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
import { classifyListenerFailure } from './listener-severity';
import {
  isListenerReconnectAbort,
  isListenerShutdownAbort,
  LISTENER_RECONNECT_REASON,
  LISTENER_SHUTDOWN_REASON,
} from './listener-abort';
import { WorkerStatus, workerStatus } from '@hatchet/v1/client/worker/health-server';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5000; // milliseconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 20;
const DEFAULT_STREAM_INACTIVE_RECONNECT_COUNT = 10;
const STREAM_INACTIVE_RECONNECT_BACKOFF_MS = 3000;

enum ListenStrategy {
  LISTEN_STRATEGY_V1 = 1,
  LISTEN_STRATEGY_V2 = 2,
}

export type ActionKey = `${string}/${number}`;

export type Action = AssignedAction & { readonly key: ActionKey };

export function workflowNameFromAction(
  action: Pick<AssignedAction, 'actionId' | 'jobName'>
): string {
  const separatorIndex = action.actionId.lastIndexOf(':');
  return separatorIndex === -1 ? action.jobName : action.actionId.substring(0, separatorIndex);
}

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
  streamInactiveReconnects = 0;
  maxStreamInactiveReconnects = DEFAULT_STREAM_INACTIVE_RECONNECT_COUNT;
  reconnecting = false;
  onWorkerStatusChange?: (status: WorkerStatus) => void;

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

  setWorkerStatusCallback(callback: (status: WorkerStatus) => void): void {
    this.onWorkerStatusChange = callback;
  }

  private handleStreamInactive(): void {
    if (this.done || this.reconnecting) {
      return;
    }

    this.reconnecting = true;
    this.streamInactiveReconnects += 1;
    this.onWorkerStatusChange?.(workerStatus.UNHEALTHY);

    this.logger.warn(
      `Listener stream inactive, reconnecting (${this.streamInactiveReconnects}/${this.maxStreamInactiveReconnects})`
    );

    this.heartbeat.stop();

    if (this.abortController) {
      this.abortController.abort(LISTENER_RECONNECT_REASON);
    }
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
          if (isAbortError(e)) {
            if (isListenerReconnectAbort(e)) {
              client.reconnecting = false;

              if (client.streamInactiveReconnects > client.maxStreamInactiveReconnects) {
                throw new HatchetError(
                  `Could not re-establish listener after ${client.maxStreamInactiveReconnects} inactive-stream reconnect attempts`
                );
              }

              client.logger.warn('Listener reconnecting after inactive stream');
              await sleep(STREAM_INACTIVE_RECONNECT_BACKOFF_MS);
              continue;
            }

            if (isListenerShutdownAbort(e)) {
              client.logger.info('Listener aborted, exiting generator');
              break;
            }

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

          const message = `Listener encountered an error: ${getErrorMessage(e)}`;
          const severity = classifyListenerFailure(e, client.retries);
          if (severity !== 'silent') {
            client.logger[severity](message);
          }

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

      await this.heartbeat.start(() => this.handleStreamInactive());
      this.streamInactiveReconnects = 0;
      this.onWorkerStatusChange?.(workerStatus.HEALTHY);
      this.logger.green('Connection established using LISTEN_STRATEGY_V2');
      return res;
    } catch (e: unknown) {
      this.retries += 1;

      const message = `Attempt ${this.retries}: Failed to connect, retrying...`;
      const severity = classifyListenerFailure(e, this.retries);
      if (severity !== 'silent') {
        this.logger[severity](message);
      }

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
      this.abortController.abort(LISTENER_SHUTDOWN_REASON);
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
