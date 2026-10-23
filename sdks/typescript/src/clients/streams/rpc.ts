import { V1StreamsClient, V1StreamsDefinition } from '@hatchet/protoc/v1/streams';
import { V1Streams } from '@hatchet/protoc-es/v1/streams_pb';
import { createTsProtoClient } from '@clients/transport/ts-proto-client';
import type { Transport } from '@clients/transport/transport';

/**
 * The v1 `V1Streams` client, every RPC of the generated `V1StreamsClient` interface, served
 * by a Connect client on the given transport. `Subscribe` is a server stream, which the unary
 * transport does not serve: calling it throws.
 */
export function createV1StreamsRpc(transport: Transport): V1StreamsClient {
  return createTsProtoClient(V1StreamsDefinition, V1Streams, transport);
}
