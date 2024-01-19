// package: 
// file: workflows/workflows.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as workflows_workflows_pb from "../workflows/workflows_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_wrappers_pb from "google-protobuf/google/protobuf/wrappers_pb";

interface IWorkflowServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    listWorkflows: IWorkflowServiceService_IListWorkflows;
    putWorkflow: IWorkflowServiceService_IPutWorkflow;
    scheduleWorkflow: IWorkflowServiceService_IScheduleWorkflow;
    getWorkflowByName: IWorkflowServiceService_IGetWorkflowByName;
    listWorkflowsForEvent: IWorkflowServiceService_IListWorkflowsForEvent;
    deleteWorkflow: IWorkflowServiceService_IDeleteWorkflow;
}

interface IWorkflowServiceService_IListWorkflows extends grpc.MethodDefinition<workflows_workflows_pb.ListWorkflowsRequest, workflows_workflows_pb.ListWorkflowsResponse> {
    path: "/WorkflowService/ListWorkflows";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.ListWorkflowsRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.ListWorkflowsRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.ListWorkflowsResponse>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.ListWorkflowsResponse>;
}
interface IWorkflowServiceService_IPutWorkflow extends grpc.MethodDefinition<workflows_workflows_pb.PutWorkflowRequest, workflows_workflows_pb.WorkflowVersion> {
    path: "/WorkflowService/PutWorkflow";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.PutWorkflowRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.PutWorkflowRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.WorkflowVersion>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.WorkflowVersion>;
}
interface IWorkflowServiceService_IScheduleWorkflow extends grpc.MethodDefinition<workflows_workflows_pb.ScheduleWorkflowRequest, workflows_workflows_pb.WorkflowVersion> {
    path: "/WorkflowService/ScheduleWorkflow";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.ScheduleWorkflowRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.ScheduleWorkflowRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.WorkflowVersion>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.WorkflowVersion>;
}
interface IWorkflowServiceService_IGetWorkflowByName extends grpc.MethodDefinition<workflows_workflows_pb.GetWorkflowByNameRequest, workflows_workflows_pb.Workflow> {
    path: "/WorkflowService/GetWorkflowByName";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.GetWorkflowByNameRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.GetWorkflowByNameRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.Workflow>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.Workflow>;
}
interface IWorkflowServiceService_IListWorkflowsForEvent extends grpc.MethodDefinition<workflows_workflows_pb.ListWorkflowsForEventRequest, workflows_workflows_pb.ListWorkflowsResponse> {
    path: "/WorkflowService/ListWorkflowsForEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.ListWorkflowsForEventRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.ListWorkflowsForEventRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.ListWorkflowsResponse>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.ListWorkflowsResponse>;
}
interface IWorkflowServiceService_IDeleteWorkflow extends grpc.MethodDefinition<workflows_workflows_pb.DeleteWorkflowRequest, workflows_workflows_pb.Workflow> {
    path: "/WorkflowService/DeleteWorkflow";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<workflows_workflows_pb.DeleteWorkflowRequest>;
    requestDeserialize: grpc.deserialize<workflows_workflows_pb.DeleteWorkflowRequest>;
    responseSerialize: grpc.serialize<workflows_workflows_pb.Workflow>;
    responseDeserialize: grpc.deserialize<workflows_workflows_pb.Workflow>;
}

export const WorkflowServiceService: IWorkflowServiceService;

export interface IWorkflowServiceServer extends grpc.UntypedServiceImplementation {
    listWorkflows: grpc.handleUnaryCall<workflows_workflows_pb.ListWorkflowsRequest, workflows_workflows_pb.ListWorkflowsResponse>;
    putWorkflow: grpc.handleUnaryCall<workflows_workflows_pb.PutWorkflowRequest, workflows_workflows_pb.WorkflowVersion>;
    scheduleWorkflow: grpc.handleUnaryCall<workflows_workflows_pb.ScheduleWorkflowRequest, workflows_workflows_pb.WorkflowVersion>;
    getWorkflowByName: grpc.handleUnaryCall<workflows_workflows_pb.GetWorkflowByNameRequest, workflows_workflows_pb.Workflow>;
    listWorkflowsForEvent: grpc.handleUnaryCall<workflows_workflows_pb.ListWorkflowsForEventRequest, workflows_workflows_pb.ListWorkflowsResponse>;
    deleteWorkflow: grpc.handleUnaryCall<workflows_workflows_pb.DeleteWorkflowRequest, workflows_workflows_pb.Workflow>;
}

export interface IWorkflowServiceClient {
    listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
}

export class WorkflowServiceClient extends grpc.Client implements IWorkflowServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflows(request: workflows_workflows_pb.ListWorkflowsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public putWorkflow(request: workflows_workflows_pb.PutWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public scheduleWorkflow(request: workflows_workflows_pb.ScheduleWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.WorkflowVersion) => void): grpc.ClientUnaryCall;
    public getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    public getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    public getWorkflowByName(request: workflows_workflows_pb.GetWorkflowByNameRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    public listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public listWorkflowsForEvent(request: workflows_workflows_pb.ListWorkflowsForEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.ListWorkflowsResponse) => void): grpc.ClientUnaryCall;
    public deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    public deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
    public deleteWorkflow(request: workflows_workflows_pb.DeleteWorkflowRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: workflows_workflows_pb.Workflow) => void): grpc.ClientUnaryCall;
}
