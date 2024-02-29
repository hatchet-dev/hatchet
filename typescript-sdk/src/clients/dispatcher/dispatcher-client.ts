import { Channel, ClientFactory } from 'nice-grpc';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  StepActionEvent,
  GroupKeyActionEvent,
  OverridesData,
  DeepPartial,
} from '@hatchet/protoc/dispatcher';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';

import { retrier } from '@hatchet/util/retrier';
import { ActionListener } from './action-listener';

interface GetActionListenerOptions {
  workerName: string;
  services: string[];
  actions: string[];
  maxRuns?: number;
}

export class DispatcherClient {
  config: ClientConfig;
  client: PbDispatcherClient;
  logger: Logger;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
    this.logger = new Logger(`Dispatcher`, config.log_level);
  }

  async getActionListener(options: GetActionListenerOptions) {
    // Register the worker
    const registration = await this.client.register({
      ...options,
    });

    return new ActionListener(this, registration.workerId);
  }

  async sendStepActionEvent(in_: StepActionEvent) {
    try {
      return retrier(async () => this.client.sendStepActionEvent(in_), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async sendGroupKeyActionEvent(in_: GroupKeyActionEvent) {
    try {
      return retrier(async () => this.client.sendGroupKeyActionEvent(in_), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  async putOverridesData(in_: DeepPartial<OverridesData>) {
    try {
      return retrier(async () => this.client.putOverridesData(in_), this.logger);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
