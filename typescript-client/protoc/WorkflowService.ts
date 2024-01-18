// Original file: ../api-contracts/workflows/workflows.proto

import type * as grpc from '@grpc/grpc-js'
import type { MethodDefinition } from '@grpc/proto-loader'
import type { DeleteWorkflowRequest as _DeleteWorkflowRequest, DeleteWorkflowRequest__Output as _DeleteWorkflowRequest__Output } from './DeleteWorkflowRequest';
import type { GetWorkflowByNameRequest as _GetWorkflowByNameRequest, GetWorkflowByNameRequest__Output as _GetWorkflowByNameRequest__Output } from './GetWorkflowByNameRequest';
import type { ListWorkflowsForEventRequest as _ListWorkflowsForEventRequest, ListWorkflowsForEventRequest__Output as _ListWorkflowsForEventRequest__Output } from './ListWorkflowsForEventRequest';
import type { ListWorkflowsRequest as _ListWorkflowsRequest, ListWorkflowsRequest__Output as _ListWorkflowsRequest__Output } from './ListWorkflowsRequest';
import type { ListWorkflowsResponse as _ListWorkflowsResponse, ListWorkflowsResponse__Output as _ListWorkflowsResponse__Output } from './ListWorkflowsResponse';
import type { PutWorkflowRequest as _PutWorkflowRequest, PutWorkflowRequest__Output as _PutWorkflowRequest__Output } from './PutWorkflowRequest';
import type { ScheduleWorkflowRequest as _ScheduleWorkflowRequest, ScheduleWorkflowRequest__Output as _ScheduleWorkflowRequest__Output } from './ScheduleWorkflowRequest';
import type { Workflow as _Workflow, Workflow__Output as _Workflow__Output } from './Workflow';
import type { WorkflowVersion as _WorkflowVersion, WorkflowVersion__Output as _WorkflowVersion__Output } from './WorkflowVersion';

export interface WorkflowServiceClient extends grpc.Client {
  DeleteWorkflow(argument: _DeleteWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  DeleteWorkflow(argument: _DeleteWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  DeleteWorkflow(argument: _DeleteWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  DeleteWorkflow(argument: _DeleteWorkflowRequest, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  deleteWorkflow(argument: _DeleteWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  deleteWorkflow(argument: _DeleteWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  deleteWorkflow(argument: _DeleteWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  deleteWorkflow(argument: _DeleteWorkflowRequest, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  
  GetWorkflowByName(argument: _GetWorkflowByNameRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  GetWorkflowByName(argument: _GetWorkflowByNameRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  GetWorkflowByName(argument: _GetWorkflowByNameRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  GetWorkflowByName(argument: _GetWorkflowByNameRequest, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  getWorkflowByName(argument: _GetWorkflowByNameRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  getWorkflowByName(argument: _GetWorkflowByNameRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  getWorkflowByName(argument: _GetWorkflowByNameRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  getWorkflowByName(argument: _GetWorkflowByNameRequest, callback: grpc.requestCallback<_Workflow__Output>): grpc.ClientUnaryCall;
  
  ListWorkflows(argument: _ListWorkflowsRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflows(argument: _ListWorkflowsRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflows(argument: _ListWorkflowsRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflows(argument: _ListWorkflowsRequest, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflows(argument: _ListWorkflowsRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflows(argument: _ListWorkflowsRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflows(argument: _ListWorkflowsRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflows(argument: _ListWorkflowsRequest, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  
  ListWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  ListWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  listWorkflowsForEvent(argument: _ListWorkflowsForEventRequest, callback: grpc.requestCallback<_ListWorkflowsResponse__Output>): grpc.ClientUnaryCall;
  
  PutWorkflow(argument: _PutWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  PutWorkflow(argument: _PutWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  PutWorkflow(argument: _PutWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  PutWorkflow(argument: _PutWorkflowRequest, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  putWorkflow(argument: _PutWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  putWorkflow(argument: _PutWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  putWorkflow(argument: _PutWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  putWorkflow(argument: _PutWorkflowRequest, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  
  ScheduleWorkflow(argument: _ScheduleWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  ScheduleWorkflow(argument: _ScheduleWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  ScheduleWorkflow(argument: _ScheduleWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  ScheduleWorkflow(argument: _ScheduleWorkflowRequest, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  scheduleWorkflow(argument: _ScheduleWorkflowRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  scheduleWorkflow(argument: _ScheduleWorkflowRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  scheduleWorkflow(argument: _ScheduleWorkflowRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  scheduleWorkflow(argument: _ScheduleWorkflowRequest, callback: grpc.requestCallback<_WorkflowVersion__Output>): grpc.ClientUnaryCall;
  
}

export interface WorkflowServiceHandlers extends grpc.UntypedServiceImplementation {
  DeleteWorkflow: grpc.handleUnaryCall<_DeleteWorkflowRequest__Output, _Workflow>;
  
  GetWorkflowByName: grpc.handleUnaryCall<_GetWorkflowByNameRequest__Output, _Workflow>;
  
  ListWorkflows: grpc.handleUnaryCall<_ListWorkflowsRequest__Output, _ListWorkflowsResponse>;
  
  ListWorkflowsForEvent: grpc.handleUnaryCall<_ListWorkflowsForEventRequest__Output, _ListWorkflowsResponse>;
  
  PutWorkflow: grpc.handleUnaryCall<_PutWorkflowRequest__Output, _WorkflowVersion>;
  
  ScheduleWorkflow: grpc.handleUnaryCall<_ScheduleWorkflowRequest__Output, _WorkflowVersion>;
  
}

export interface WorkflowServiceDefinition extends grpc.ServiceDefinition {
  DeleteWorkflow: MethodDefinition<_DeleteWorkflowRequest, _Workflow, _DeleteWorkflowRequest__Output, _Workflow__Output>
  GetWorkflowByName: MethodDefinition<_GetWorkflowByNameRequest, _Workflow, _GetWorkflowByNameRequest__Output, _Workflow__Output>
  ListWorkflows: MethodDefinition<_ListWorkflowsRequest, _ListWorkflowsResponse, _ListWorkflowsRequest__Output, _ListWorkflowsResponse__Output>
  ListWorkflowsForEvent: MethodDefinition<_ListWorkflowsForEventRequest, _ListWorkflowsResponse, _ListWorkflowsForEventRequest__Output, _ListWorkflowsResponse__Output>
  PutWorkflow: MethodDefinition<_PutWorkflowRequest, _WorkflowVersion, _PutWorkflowRequest__Output, _WorkflowVersion__Output>
  ScheduleWorkflow: MethodDefinition<_ScheduleWorkflowRequest, _WorkflowVersion, _ScheduleWorkflowRequest__Output, _WorkflowVersion__Output>
}
