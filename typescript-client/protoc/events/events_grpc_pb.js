// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var events_events_pb = require('../events/events_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_Event(arg) {
  if (!(arg instanceof events_events_pb.Event)) {
    throw new Error('Expected argument of type Event');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Event(buffer_arg) {
  return events_events_pb.Event.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ListEventRequest(arg) {
  if (!(arg instanceof events_events_pb.ListEventRequest)) {
    throw new Error('Expected argument of type ListEventRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ListEventRequest(buffer_arg) {
  return events_events_pb.ListEventRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ListEventResponse(arg) {
  if (!(arg instanceof events_events_pb.ListEventResponse)) {
    throw new Error('Expected argument of type ListEventResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ListEventResponse(buffer_arg) {
  return events_events_pb.ListEventResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_PushEventRequest(arg) {
  if (!(arg instanceof events_events_pb.PushEventRequest)) {
    throw new Error('Expected argument of type PushEventRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_PushEventRequest(buffer_arg) {
  return events_events_pb.PushEventRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_ReplayEventRequest(arg) {
  if (!(arg instanceof events_events_pb.ReplayEventRequest)) {
    throw new Error('Expected argument of type ReplayEventRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_ReplayEventRequest(buffer_arg) {
  return events_events_pb.ReplayEventRequest.deserializeBinary(new Uint8Array(buffer_arg));
}


var EventsServiceService = exports.EventsServiceService = {
  push: {
    path: '/EventsService/Push',
    requestStream: false,
    responseStream: false,
    requestType: events_events_pb.PushEventRequest,
    responseType: events_events_pb.Event,
    requestSerialize: serialize_PushEventRequest,
    requestDeserialize: deserialize_PushEventRequest,
    responseSerialize: serialize_Event,
    responseDeserialize: deserialize_Event,
  },
  list: {
    path: '/EventsService/List',
    requestStream: false,
    responseStream: false,
    requestType: events_events_pb.ListEventRequest,
    responseType: events_events_pb.ListEventResponse,
    requestSerialize: serialize_ListEventRequest,
    requestDeserialize: deserialize_ListEventRequest,
    responseSerialize: serialize_ListEventResponse,
    responseDeserialize: deserialize_ListEventResponse,
  },
  replaySingleEvent: {
    path: '/EventsService/ReplaySingleEvent',
    requestStream: false,
    responseStream: false,
    requestType: events_events_pb.ReplayEventRequest,
    responseType: events_events_pb.Event,
    requestSerialize: serialize_ReplayEventRequest,
    requestDeserialize: deserialize_ReplayEventRequest,
    responseSerialize: serialize_Event,
    responseDeserialize: deserialize_Event,
  },
};

exports.EventsServiceClient = grpc.makeGenericClientConstructor(EventsServiceService);
