import type * as grpc from '@grpc/grpc-js';
import type { MessageTypeDefinition } from '@grpc/proto-loader';

import type { WorkflowServiceClient as _WorkflowServiceClient, WorkflowServiceDefinition as _WorkflowServiceDefinition } from './WorkflowService';

type SubtypeConstructor<Constructor extends new (...args: any) => any, Subtype> = {
  new(...args: ConstructorParameters<Constructor>): Subtype;
};

export interface ProtoGrpcType {
  CreateWorkflowJobOpts: MessageTypeDefinition
  CreateWorkflowStepOpts: MessageTypeDefinition
  CreateWorkflowVersionOpts: MessageTypeDefinition
  DeleteWorkflowRequest: MessageTypeDefinition
  GetWorkflowByNameRequest: MessageTypeDefinition
  Job: MessageTypeDefinition
  ListWorkflowsForEventRequest: MessageTypeDefinition
  ListWorkflowsRequest: MessageTypeDefinition
  ListWorkflowsResponse: MessageTypeDefinition
  PutWorkflowRequest: MessageTypeDefinition
  ScheduleWorkflowRequest: MessageTypeDefinition
  Step: MessageTypeDefinition
  Workflow: MessageTypeDefinition
  WorkflowService: SubtypeConstructor<typeof grpc.Client, _WorkflowServiceClient> & { service: _WorkflowServiceDefinition }
  WorkflowTriggerCronRef: MessageTypeDefinition
  WorkflowTriggerEventRef: MessageTypeDefinition
  WorkflowTriggers: MessageTypeDefinition
  WorkflowVersion: MessageTypeDefinition
  google: {
    protobuf: {
      BoolValue: MessageTypeDefinition
      BytesValue: MessageTypeDefinition
      DoubleValue: MessageTypeDefinition
      FloatValue: MessageTypeDefinition
      Int32Value: MessageTypeDefinition
      Int64Value: MessageTypeDefinition
      StringValue: MessageTypeDefinition
      Timestamp: MessageTypeDefinition
      UInt32Value: MessageTypeDefinition
      UInt64Value: MessageTypeDefinition
    }
  }
}

