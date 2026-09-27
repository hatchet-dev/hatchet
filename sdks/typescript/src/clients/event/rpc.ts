import { EventsServiceClient, EventsServiceDefinition } from '@hatchet/protoc/events/events';
import { EventsService } from '@hatchet/protoc-es/events/events_pb';
import { createTsProtoClient, type Transport } from '@clients/transport';

/**
 * The `EventsService` client, every RPC of the generated `EventsServiceClient` interface,
 * served by a Connect client on the given transport.
 */
export function createEventsRpc(transport: Transport): EventsServiceClient {
  return createTsProtoClient(EventsServiceDefinition, EventsService, transport);
}
