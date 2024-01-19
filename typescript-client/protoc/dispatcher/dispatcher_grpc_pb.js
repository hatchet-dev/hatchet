// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var dispatcher_dispatcher_pb = require('../dispatcher/dispatcher_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_ActionEvent(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.ActionEvent)) {
    throw new Error('Expected argument of type ActionEvent');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ActionEvent(buffer_arg) {
  return dispatcher_dispatcher_pb.ActionEvent.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ActionEventResponse(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.ActionEventResponse)) {
    throw new Error('Expected argument of type ActionEventResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ActionEventResponse(buffer_arg) {
  return dispatcher_dispatcher_pb.ActionEventResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_AssignedAction(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.AssignedAction)) {
    throw new Error('Expected argument of type AssignedAction');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_AssignedAction(buffer_arg) {
  return dispatcher_dispatcher_pb.AssignedAction.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkerListenRequest(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.WorkerListenRequest)) {
    throw new Error('Expected argument of type WorkerListenRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkerListenRequest(buffer_arg) {
  return dispatcher_dispatcher_pb.WorkerListenRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkerRegisterRequest(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.WorkerRegisterRequest)) {
    throw new Error('Expected argument of type WorkerRegisterRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkerRegisterRequest(buffer_arg) {
  return dispatcher_dispatcher_pb.WorkerRegisterRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkerRegisterResponse(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.WorkerRegisterResponse)) {
    throw new Error('Expected argument of type WorkerRegisterResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkerRegisterResponse(buffer_arg) {
  return dispatcher_dispatcher_pb.WorkerRegisterResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkerUnsubscribeRequest(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.WorkerUnsubscribeRequest)) {
    throw new Error('Expected argument of type WorkerUnsubscribeRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkerUnsubscribeRequest(buffer_arg) {
  return dispatcher_dispatcher_pb.WorkerUnsubscribeRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_WorkerUnsubscribeResponse(arg) {
  if (!(arg instanceof dispatcher_dispatcher_pb.WorkerUnsubscribeResponse)) {
    throw new Error('Expected argument of type WorkerUnsubscribeResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_WorkerUnsubscribeResponse(buffer_arg) {
  return dispatcher_dispatcher_pb.WorkerUnsubscribeResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var DispatcherService = exports.DispatcherService = {
  register: {
    path: '/Dispatcher/Register',
    requestStream: false,
    responseStream: false,
    requestType: dispatcher_dispatcher_pb.WorkerRegisterRequest,
    responseType: dispatcher_dispatcher_pb.WorkerRegisterResponse,
    requestSerialize: serialize_WorkerRegisterRequest,
    requestDeserialize: deserialize_WorkerRegisterRequest,
    responseSerialize: serialize_WorkerRegisterResponse,
    responseDeserialize: deserialize_WorkerRegisterResponse,
  },
  listen: {
    path: '/Dispatcher/Listen',
    requestStream: false,
    responseStream: true,
    requestType: dispatcher_dispatcher_pb.WorkerListenRequest,
    responseType: dispatcher_dispatcher_pb.AssignedAction,
    requestSerialize: serialize_WorkerListenRequest,
    requestDeserialize: deserialize_WorkerListenRequest,
    responseSerialize: serialize_AssignedAction,
    responseDeserialize: deserialize_AssignedAction,
  },
  sendActionEvent: {
    path: '/Dispatcher/SendActionEvent',
    requestStream: false,
    responseStream: false,
    requestType: dispatcher_dispatcher_pb.ActionEvent,
    responseType: dispatcher_dispatcher_pb.ActionEventResponse,
    requestSerialize: serialize_ActionEvent,
    requestDeserialize: deserialize_ActionEvent,
    responseSerialize: serialize_ActionEventResponse,
    responseDeserialize: deserialize_ActionEventResponse,
  },
  unsubscribe: {
    path: '/Dispatcher/Unsubscribe',
    requestStream: false,
    responseStream: false,
    requestType: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest,
    responseType: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse,
    requestSerialize: serialize_WorkerUnsubscribeRequest,
    requestDeserialize: deserialize_WorkerUnsubscribeRequest,
    responseSerialize: serialize_WorkerUnsubscribeResponse,
    responseDeserialize: deserialize_WorkerUnsubscribeResponse,
  },
};

exports.DispatcherClient = grpc.makeGenericClientConstructor(DispatcherService);
