import { Client, createClient } from '@connectrpc/connect';
import {
  BulkTriggerWorkflowRequest,
  BulkTriggerWorkflowResponse,
  PutRateLimitRequest,
  PutRateLimitResponse,
  PutWorkflowRequest,
  ScheduleWorkflowRequest,
  TriggerWorkflowResponse,
  WorkflowVersion,
} from '@hatchet/protoc/workflows/workflows';
import { TriggerWorkflowRequest } from '@hatchet/protoc/v1/shared/trigger';
import { TriggerWorkflowRequestSchema } from '@hatchet/protoc-es/v1/shared/trigger_pb';
import {
  BulkTriggerWorkflowRequestSchema,
  BulkTriggerWorkflowResponseSchema,
  PutRateLimitRequestSchema,
  PutRateLimitResponseSchema,
  PutWorkflowRequestSchema,
  ScheduleWorkflowRequestSchema,
  TriggerWorkflowResponseSchema,
  WorkflowService,
  WorkflowVersionSchema,
} from '@hatchet/protoc-es/workflows/workflows_pb';
import {
  CreateWorkflowVersionRequest,
  CreateWorkflowVersionResponse,
} from '@hatchet/protoc/v1/workflows';
import {
  AdminService,
  CreateWorkflowVersionRequestSchema,
  CreateWorkflowVersionResponseSchema,
} from '@hatchet/protoc-es/v1/workflows_pb';
import { fromProtobufEs, toProtobufEs, type DeepPartial, type Transport } from '@clients/transport';

/**
 * The `WorkflowService` RPCs the SDK calls, typed with the SDK's message types and served by a
 * Connect client on the given transport.
 */
export interface WorkflowsRpc {
  putWorkflow(request: DeepPartial<PutWorkflowRequest>): Promise<WorkflowVersion>;
  scheduleWorkflow(request: DeepPartial<ScheduleWorkflowRequest>): Promise<WorkflowVersion>;
  triggerWorkflow(request: DeepPartial<TriggerWorkflowRequest>): Promise<TriggerWorkflowResponse>;
  bulkTriggerWorkflow(
    request: DeepPartial<BulkTriggerWorkflowRequest>
  ): Promise<BulkTriggerWorkflowResponse>;
  putRateLimit(request: DeepPartial<PutRateLimitRequest>): Promise<PutRateLimitResponse>;
}

export function createWorkflowsRpc(transport: Transport): WorkflowsRpc {
  const client: Client<typeof WorkflowService> = createClient(WorkflowService, transport);

  return {
    putWorkflow: async (request) =>
      fromProtobufEs(
        WorkflowVersion,
        WorkflowVersionSchema,
        await client.putWorkflow(
          toProtobufEs(PutWorkflowRequestSchema, PutWorkflowRequest, request)
        )
      ),
    scheduleWorkflow: async (request) =>
      fromProtobufEs(
        WorkflowVersion,
        WorkflowVersionSchema,
        await client.scheduleWorkflow(
          toProtobufEs(ScheduleWorkflowRequestSchema, ScheduleWorkflowRequest, request)
        )
      ),
    triggerWorkflow: async (request) =>
      fromProtobufEs(
        TriggerWorkflowResponse,
        TriggerWorkflowResponseSchema,
        await client.triggerWorkflow(
          toProtobufEs(TriggerWorkflowRequestSchema, TriggerWorkflowRequest, request)
        )
      ),
    bulkTriggerWorkflow: async (request) =>
      fromProtobufEs(
        BulkTriggerWorkflowResponse,
        BulkTriggerWorkflowResponseSchema,
        await client.bulkTriggerWorkflow(
          toProtobufEs(BulkTriggerWorkflowRequestSchema, BulkTriggerWorkflowRequest, request)
        )
      ),
    putRateLimit: async (request) =>
      fromProtobufEs(
        PutRateLimitResponse,
        PutRateLimitResponseSchema,
        await client.putRateLimit(
          toProtobufEs(PutRateLimitRequestSchema, PutRateLimitRequest, request)
        )
      ),
  };
}

/**
 * The v1 `AdminService` RPCs the admin clients call, typed with the SDK's message types and
 * served by a Connect client on the given transport.
 */
export interface V1AdminRpc {
  putWorkflow(
    request: DeepPartial<CreateWorkflowVersionRequest>
  ): Promise<CreateWorkflowVersionResponse>;
}

export function createV1AdminRpc(transport: Transport): V1AdminRpc {
  const client: Client<typeof AdminService> = createClient(AdminService, transport);

  return {
    putWorkflow: async (request) =>
      fromProtobufEs(
        CreateWorkflowVersionResponse,
        CreateWorkflowVersionResponseSchema,
        await client.putWorkflow(
          toProtobufEs(CreateWorkflowVersionRequestSchema, CreateWorkflowVersionRequest, request)
        )
      ),
  };
}
