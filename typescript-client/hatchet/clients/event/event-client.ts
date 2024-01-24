import { createChannel, createClient } from 'nice-grpc';
import {
  EventsServiceClient,
  EventsServiceDefinition,
  PushEventRequest,
} from '@protoc/events/events';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  constructor(config: ClientConfig) {
    this.config = config;
    this.client = createClient(
      EventsServiceDefinition,
      createChannel(
        config.host_port
        // FIXME: Credentials ChannelCredentials.createSsl(config.tls_config.)
      )
    );
  }

  push<T>(type: string, input: T) {
    const req: PushEventRequest = {
      tenantId: this.config.tenant_id,
      key: type,
      payload: JSON.stringify(input),
      eventTimestamp: new Date(),
    };

    try {
      return this.client.push(req);
    } catch (e: any) {
      // FIXME: any
      throw new HatchetError(e.message);
    }
  }
}
