// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var workflows_workflows_pb = require('../workflows/workflows_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
var google_protobuf_wrappers_pb = require('google-protobuf/google/protobuf/wrappers_pb.js');

function serialize_DeleteWorkflowRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.DeleteWorkflowRequest)) {
    throw new Error('Expected argument of type DeleteWorkflowRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_DeleteWorkflowRequest(buffer_arg) {
  return workflows_workflows_pb.DeleteWorkflowRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_GetWorkflowByNameRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.GetWorkflowByNameRequest)) {
    throw new Error('Expected argument of type GetWorkflowByNameRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_GetWorkflowByNameRequest(buffer_arg) {
  return workflows_workflows_pb.GetWorkflowByNameRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ListWorkflowsForEventRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.ListWorkflowsForEventRequest)) {
    throw new Error('Expected argument of type ListWorkflowsForEventRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ListWorkflowsForEventRequest(buffer_arg) {
  return workflows_workflows_pb.ListWorkflowsForEventRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ListWorkflowsRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.ListWorkflowsRequest)) {
    throw new Error('Expected argument of type ListWorkflowsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ListWorkflowsRequest(buffer_arg) {
  return workflows_workflows_pb.ListWorkflowsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ListWorkflowsResponse(arg) {
  if (!(arg instanceof workflows_workflows_pb.ListWorkflowsResponse)) {
    throw new Error('Expected argument of type ListWorkflowsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ListWorkflowsResponse(buffer_arg) {
  return workflows_workflows_pb.ListWorkflowsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_PutWorkflowRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.PutWorkflowRequest)) {
    throw new Error('Expected argument of type PutWorkflowRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_PutWorkflowRequest(buffer_arg) {
  return workflows_workflows_pb.PutWorkflowRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ScheduleWorkflowRequest(arg) {
  if (!(arg instanceof workflows_workflows_pb.ScheduleWorkflowRequest)) {
    throw new Error('Expected argument of type ScheduleWorkflowRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ScheduleWorkflowRequest(buffer_arg) {
  return workflows_workflows_pb.ScheduleWorkflowRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_Workflow(arg) {
  if (!(arg instanceof workflows_workflows_pb.Workflow)) {
    throw new Error('Expected argument of type Workflow');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Workflow(buffer_arg) {
  return workflows_workflows_pb.Workflow.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkflowVersion(arg) {
  if (!(arg instanceof workflows_workflows_pb.WorkflowVersion)) {
    throw new Error('Expected argument of type WorkflowVersion');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkflowVersion(buffer_arg) {
  return workflows_workflows_pb.WorkflowVersion.deserializeBinary(new Uint8Array(buffer_arg));
}


// WorkflowService represents a set of RPCs for managing workflows.
var WorkflowServiceService = exports.WorkflowServiceService = {
  listWorkflows: {
    path: '/WorkflowService/ListWorkflows',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.ListWorkflowsRequest,
    responseType: workflows_workflows_pb.ListWorkflowsResponse,
    requestSerialize: serialize_ListWorkflowsRequest,
    requestDeserialize: deserialize_ListWorkflowsRequest,
    responseSerialize: serialize_ListWorkflowsResponse,
    responseDeserialize: deserialize_ListWorkflowsResponse,
  },
  putWorkflow: {
    path: '/WorkflowService/PutWorkflow',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.PutWorkflowRequest,
    responseType: workflows_workflows_pb.WorkflowVersion,
    requestSerialize: serialize_PutWorkflowRequest,
    requestDeserialize: deserialize_PutWorkflowRequest,
    responseSerialize: serialize_WorkflowVersion,
    responseDeserialize: deserialize_WorkflowVersion,
  },
  scheduleWorkflow: {
    path: '/WorkflowService/ScheduleWorkflow',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.ScheduleWorkflowRequest,
    responseType: workflows_workflows_pb.WorkflowVersion,
    requestSerialize: serialize_ScheduleWorkflowRequest,
    requestDeserialize: deserialize_ScheduleWorkflowRequest,
    responseSerialize: serialize_WorkflowVersion,
    responseDeserialize: deserialize_WorkflowVersion,
  },
  getWorkflowByName: {
    path: '/WorkflowService/GetWorkflowByName',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.GetWorkflowByNameRequest,
    responseType: workflows_workflows_pb.Workflow,
    requestSerialize: serialize_GetWorkflowByNameRequest,
    requestDeserialize: deserialize_GetWorkflowByNameRequest,
    responseSerialize: serialize_Workflow,
    responseDeserialize: deserialize_Workflow,
  },
  listWorkflowsForEvent: {
    path: '/WorkflowService/ListWorkflowsForEvent',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.ListWorkflowsForEventRequest,
    responseType: workflows_workflows_pb.ListWorkflowsResponse,
    requestSerialize: serialize_ListWorkflowsForEventRequest,
    requestDeserialize: deserialize_ListWorkflowsForEventRequest,
    responseSerialize: serialize_ListWorkflowsResponse,
    responseDeserialize: deserialize_ListWorkflowsResponse,
  },
  deleteWorkflow: {
    path: '/WorkflowService/DeleteWorkflow',
    requestStream: false,
    responseStream: false,
    requestType: workflows_workflows_pb.DeleteWorkflowRequest,
    responseType: workflows_workflows_pb.Workflow,
    requestSerialize: serialize_DeleteWorkflowRequest,
    requestDeserialize: deserialize_DeleteWorkflowRequest,
    responseSerialize: serialize_Workflow,
    responseDeserialize: deserialize_Workflow,
  },
};

exports.WorkflowServiceClient = grpc.makeGenericClientConstructor(WorkflowServiceService);
