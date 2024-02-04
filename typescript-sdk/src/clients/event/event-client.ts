import { Channel, ClientFactory } from 'nice-grpc';
import {
  EventsServiceClient,
  EventsServiceDefinition,
  PushEventRequest,
} from '@hatchet/protoc/events/events';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  logger: Logger;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(EventsServiceDefinition, channel);
    this.logger = new Logger(`Dispatcher`, config.log_level);
  }

  push<T>(type: string, input: T) {
    const req: PushEventRequest = {
      key: type,
      payload: JSON.stringify(input),
      eventTimestamp: new Date(),
    };

    try {
      const e = this.client.push(req);
      this.logger.info(`Event pushed: ${type}`);
      return e;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
