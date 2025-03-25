// eslint-disable-next-line max-classes-per-file
import { Channel, ClientFactory } from 'nice-grpc';

import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';
import { V1DispatcherClient, V1DispatcherDefinition } from '@hatchet/protoc/v1/dispatcher';
import { SleepMatchCondition, UserEventMatchCondition } from '@hatchet/protoc/v1/shared/condition';
import { Api } from '../../rest';
import { DurableEventGrpcPooledListener } from './pooled-durable-listener-client';

export class DurableListenerClient {
  config: ClientConfig;
  client: V1DispatcherClient;
  logger: Logger;
  api: Api;

  pooledListener: DurableEventGrpcPooledListener | undefined;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory, api: Api) {
    this.config = config;
    this.client = factory.create(V1DispatcherDefinition, channel);
    this.logger = config.logger(`Listener`, config.log_level);
    this.api = api;
  }

  subscribe(request: { taskId: string; signalKey: string }) {
    if (!this.pooledListener) {
      this.pooledListener = new DurableEventGrpcPooledListener(this, () => {
        this.pooledListener = undefined;
      });
    }

    return this.pooledListener.subscribe(request);
  }

  registerDurableEvent(request: {
    taskId: string;
    signalKey: string;
    sleepConditions: Array<SleepMatchCondition>;
    userEventConditions: Array<UserEventMatchCondition>;
  }) {
    if (!this.pooledListener) {
      this.pooledListener = new DurableEventGrpcPooledListener(this, () => {
        this.pooledListener = undefined;
      });
    }

    return this.pooledListener.registerDurableEvent(request);
  }
}
