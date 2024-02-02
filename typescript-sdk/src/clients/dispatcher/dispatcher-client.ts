import { Channel, ClientFactory } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  StepActionEvent,
  GroupKeyActionEvent,
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

  async getActionListener(options: GetActionListenerOptions) {
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

  async sendStepActionEvent(in_: StepActionEvent) {
    try {
      return this.client.sendStepActionEvent(in_);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async sendGroupKeyActionEvent(in_: GroupKeyActionEvent) {
    try {
      return this.client.sendGroupKeyActionEvent(in_);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
