import { DispatcherClient, DispatcherDefinition } from '@hatchet/protoc/dispatcher';
import { Dispatcher } from '@hatchet/protoc-es/dispatcher/dispatcher_pb';
import { createTsProtoClient } from '@clients/transport/ts-proto-client';
import type { Transport } from '@clients/transport/transport';

/**
 * The `Dispatcher` client, every RPC of the generated `DispatcherClient` interface, served by
 * a Connect client on the given transport. Only its unary RPCs can be called this way; the
 * streaming ones (`Listen`, `SubscribeToWorkflowRuns`) stay on the Node worker's gRPC channel.
 */
export function createDispatcherRpc(transport: Transport): DispatcherClient {
  return createTsProtoClient(DispatcherDefinition, Dispatcher, transport);
}
