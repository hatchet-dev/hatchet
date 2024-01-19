import * as grpc from '@grpc/grpc-js';
import { EventsServiceClient } from '@protoc/events/events_grpc_pb';
import { Event, PushEventRequest } from '@protoc/events/events_pb';
import { ClientConfig } from '../client/client-config';

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  constructor(config: ClientConfig) {
    this.config = config;

    this.client = new EventsServiceClient(config.host_port, grpc.credentials.createInsecure());
  }

  push<T>(type: string, input: T) {
    return new Promise<Event>((resolve, reject) => {
      const req = new PushEventRequest();

      req.setTenantid(this.config.tenant_id);
      req.setKey(type);
      req.setPayload(JSON.stringify(input));

      this.client.push(req, (err, response) => {
        if (err) {
          reject(err);
        } else {
          resolve(response);
        }
      });
    });
  }
}
