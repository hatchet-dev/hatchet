import { Channel, ClientFactory } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  ActionEvent,
} from '@hatchet/protoc/dispatcher';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import HatchetError from '@util/errors/hatchet-error';
import { ActionListener } from './action-listener';

interface GetActionListenerOptions {
  workerName: string;
  services: string[];
  actions: string[];
}

export class DispatcherClient {
  config: ClientConfig;
  client: PbDispatcherClient;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
  }

  async get_action_listener(options: GetActionListenerOptions) {
    // Register the worker
    const registration = await this.client.register({
      ...options,
    });

    // Subscribe to the worker
    const listener = this.client.listen({
      workerId: registration.workerId,
    });

    return new ActionListener(this, listener, registration.workerId);
  }

  async send_action_event(in_: ActionEvent) {
    try {
      return this.client.sendActionEvent(in_);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
