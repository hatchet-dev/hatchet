import * as grpc from '@grpc/grpc-js';
import { EventsServiceClient } from '@protoc/events/events_grpc_pb';
import { ClientConfig } from '../client/client-config';

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  constructor(config: ClientConfig) {
    this.config = config;

    this.client = new EventsServiceClient(config.host_port, grpc.credentials.createInsecure());
  }

  push<T>(type: string, input: T) {
    // this.client.push();
    throw new Error('not implemented');
  }
}
