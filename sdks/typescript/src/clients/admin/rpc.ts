import {
  WorkflowServiceClient,
  WorkflowServiceDefinition,
} from '@hatchet/protoc/workflows/workflows';
import { WorkflowService } from '@hatchet/protoc-es/workflows/workflows_pb';
import { AdminServiceClient, AdminServiceDefinition } from '@hatchet/protoc/v1/workflows';
import { AdminService } from '@hatchet/protoc-es/v1/workflows_pb';
import { createTsProtoClient } from '@clients/transport/ts-proto-client';
import type { Transport } from '@clients/transport/transport';

/**
 * The `WorkflowService` client, every RPC of the generated `WorkflowServiceClient` interface,
 * served by a Connect client on the given transport.
 */
export function createWorkflowsRpc(transport: Transport): WorkflowServiceClient {
  return createTsProtoClient(WorkflowServiceDefinition, WorkflowService, transport);
}

/**
 * The v1 `AdminService` client, every RPC of the generated `AdminServiceClient` interface,
 * served by a Connect client on the given transport.
 */
export function createV1AdminRpc(transport: Transport): AdminServiceClient {
  return createTsProtoClient(AdminServiceDefinition, AdminService, transport);
}
