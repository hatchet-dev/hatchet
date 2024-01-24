import { createChannel, createClient } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  ActionEvent,
} from '@protoc/dispatcher';
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

  constructor(config: ClientConfig) {
    this.config = config;
    this.client = createClient(
      DispatcherDefinition,
      createChannel(config.host_port, config.credentials)
    );
  }

  async get_action_listener(options: GetActionListenerOptions) {
    // Register the worker
    const registration = await this.client.register({
      tenantId: this.config.tenant_id,
      ...options,
    });

    // Subscribe to the worker
    const listener = this.client.listen({
      tenantId: this.config.tenant_id,
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
