import {
  DispatcherClient as PbDispatcherClient,
  AssignedAction,
  ActionEvent,
} from '@protoc/dispatcher';

import { ServerError, Status } from 'nice-grpc';
import { ClientConfig } from '@clients/client/client-config';
import sleep from '@util/sleep';
import HatchetError from '@util/errors/hatchet-error';
import { DispatcherClient } from './dispatcher-client';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5; // seconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5;

export class ActionListener {
  config: ClientConfig;
  client: PbDispatcherClient;
  listener: AsyncIterable<AssignedAction>;
  workerId: string;

  constructor(client: DispatcherClient, listener: AsyncIterable<AssignedAction>, workerId: string) {
    this.config = client.config;
    this.client = client.client;
    this.listener = listener;
    this.workerId = workerId;
  }

  // actions
  // parse_action_payload

  actions = () =>
    (async function* gen(client: ActionListener) {
      while (true) {
        try {
          for await (const assignedAction of client.listener) {
            const action: ActionEvent = {
              ...assignedAction,
              workerId: '',
              eventTimestamp: undefined,
              eventType: 0,
              eventPayload: assignedAction.actionPayload,
            };

            yield action;
          }
        } catch (e: any) {
          if (e instanceof ServerError && e.code === Status.CANCELLED) {
            break;
          }
          if (e instanceof ServerError && e.code === Status.UNAVAILABLE) {
            client.retrySubscribe();
          }
          break;
        }
      }
    })(this);

  async retrySubscribe() {
    let retries = 0;

    while (retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT) {
      try {
        await sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL);

        this.listener = this.client.listen({
          tenantId: this.config.tenant_id,
          workerId: this.workerId,
        });

        return;
      } catch (e: any) {
        retries += 1;
      }
    }

    throw new HatchetError(
      `Could not subscribe to the worker after ${DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries`
    );
  }

  async unregister() {
    try {
      return this.client.unsubscribe({
        tenantId: this.config.tenant_id,
        workerId: this.workerId,
      });
    } catch (e: any) {
      throw new HatchetError(`Failed to unsubscribe: ${e.message}`);
    }
  }
}
