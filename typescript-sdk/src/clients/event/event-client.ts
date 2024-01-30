import { Channel, ClientFactory } from 'nice-grpc';
import {
  EventsServiceClient,
  EventsServiceDefinition,
  PushEventRequest,
} from '@hatchet/protoc/events/events';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(EventsServiceDefinition, channel);
  }

  push<T>(type: string, input: T) {
    const req: PushEventRequest = {
      key: type,
      payload: JSON.stringify(input),
      eventTimestamp: new Date(),
    };

    try {
      return this.client.push(req);
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }
}
