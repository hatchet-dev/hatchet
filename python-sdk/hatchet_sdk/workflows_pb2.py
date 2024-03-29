# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: workflows.proto
# Protobuf Python Version: 4.25.0
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import timestamp_pb2 as google_dot_protobuf_dot_timestamp__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x0fworkflows.proto\x1a\x1fgoogle/protobuf/timestamp.proto\">\n\x12PutWorkflowRequest\x12(\n\x04opts\x18\x01 \x01(\x0b\x32\x1a.CreateWorkflowVersionOpts\"\xbf\x02\n\x19\x43reateWorkflowVersionOpts\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x02 \x01(\t\x12\x0f\n\x07version\x18\x03 \x01(\t\x12\x16\n\x0e\x65vent_triggers\x18\x04 \x03(\t\x12\x15\n\rcron_triggers\x18\x05 \x03(\t\x12\x36\n\x12scheduled_triggers\x18\x06 \x03(\x0b\x32\x1a.google.protobuf.Timestamp\x12$\n\x04jobs\x18\x07 \x03(\x0b\x32\x16.CreateWorkflowJobOpts\x12-\n\x0b\x63oncurrency\x18\x08 \x01(\x0b\x32\x18.WorkflowConcurrencyOpts\x12\x1d\n\x10schedule_timeout\x18\t \x01(\tH\x00\x88\x01\x01\x42\x13\n\x11_schedule_timeout\"n\n\x17WorkflowConcurrencyOpts\x12\x0e\n\x06\x61\x63tion\x18\x01 \x01(\t\x12\x10\n\x08max_runs\x18\x02 \x01(\x05\x12\x31\n\x0elimit_strategy\x18\x03 \x01(\x0e\x32\x19.ConcurrencyLimitStrategy\"s\n\x15\x43reateWorkflowJobOpts\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x02 \x01(\t\x12\x0f\n\x07timeout\x18\x03 \x01(\t\x12&\n\x05steps\x18\x04 \x03(\x0b\x32\x17.CreateWorkflowStepOpts\"\x93\x01\n\x16\x43reateWorkflowStepOpts\x12\x13\n\x0breadable_id\x18\x01 \x01(\t\x12\x0e\n\x06\x61\x63tion\x18\x02 \x01(\t\x12\x0f\n\x07timeout\x18\x03 \x01(\t\x12\x0e\n\x06inputs\x18\x04 \x01(\t\x12\x0f\n\x07parents\x18\x05 \x03(\t\x12\x11\n\tuser_data\x18\x06 \x01(\t\x12\x0f\n\x07retries\x18\x07 \x01(\x05\"\x16\n\x14ListWorkflowsRequest\"e\n\x17ScheduleWorkflowRequest\x12\x0c\n\x04name\x18\x01 \x01(\t\x12-\n\tschedules\x18\x02 \x03(\x0b\x32\x1a.google.protobuf.Timestamp\x12\r\n\x05input\x18\x03 \x01(\t\"\xb2\x01\n\x0fWorkflowVersion\x12\n\n\x02id\x18\x01 \x01(\t\x12.\n\ncreated_at\x18\x02 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\x12.\n\nupdated_at\x18\x03 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\x12\x0f\n\x07version\x18\x05 \x01(\t\x12\r\n\x05order\x18\x06 \x01(\x05\x12\x13\n\x0bworkflow_id\x18\x07 \x01(\t\"?\n\x17WorkflowTriggerEventRef\x12\x11\n\tparent_id\x18\x01 \x01(\t\x12\x11\n\tevent_key\x18\x02 \x01(\t\"9\n\x16WorkflowTriggerCronRef\x12\x11\n\tparent_id\x18\x01 \x01(\t\x12\x0c\n\x04\x63ron\x18\x02 \x01(\t\"5\n\x16TriggerWorkflowRequest\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\r\n\x05input\x18\x02 \x01(\t\"2\n\x17TriggerWorkflowResponse\x12\x17\n\x0fworkflow_run_id\x18\x01 \x01(\t*l\n\x18\x43oncurrencyLimitStrategy\x12\x16\n\x12\x43\x41NCEL_IN_PROGRESS\x10\x00\x12\x0f\n\x0b\x44ROP_NEWEST\x10\x01\x12\x10\n\x0cQUEUE_NEWEST\x10\x02\x12\x15\n\x11GROUP_ROUND_ROBIN\x10\x03\x32\xcd\x01\n\x0fWorkflowService\x12\x34\n\x0bPutWorkflow\x12\x13.PutWorkflowRequest\x1a\x10.WorkflowVersion\x12>\n\x10ScheduleWorkflow\x12\x18.ScheduleWorkflowRequest\x1a\x10.WorkflowVersion\x12\x44\n\x0fTriggerWorkflow\x12\x17.TriggerWorkflowRequest\x1a\x18.TriggerWorkflowResponseBBZ@github.com/hatchet-dev/hatchet/internal/services/admin/contractsb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'workflows_pb2', _globals)
if _descriptor._USE_C_DESCRIPTORS == False:
  _globals['DESCRIPTOR']._options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z@github.com/hatchet-dev/hatchet/internal/services/admin/contracts'
  _globals['_CONCURRENCYLIMITSTRATEGY']._serialized_start=1356
  _globals['_CONCURRENCYLIMITSTRATEGY']._serialized_end=1464
  _globals['_PUTWORKFLOWREQUEST']._serialized_start=52
  _globals['_PUTWORKFLOWREQUEST']._serialized_end=114
  _globals['_CREATEWORKFLOWVERSIONOPTS']._serialized_start=117
  _globals['_CREATEWORKFLOWVERSIONOPTS']._serialized_end=436
  _globals['_WORKFLOWCONCURRENCYOPTS']._serialized_start=438
  _globals['_WORKFLOWCONCURRENCYOPTS']._serialized_end=548
  _globals['_CREATEWORKFLOWJOBOPTS']._serialized_start=550
  _globals['_CREATEWORKFLOWJOBOPTS']._serialized_end=665
  _globals['_CREATEWORKFLOWSTEPOPTS']._serialized_start=668
  _globals['_CREATEWORKFLOWSTEPOPTS']._serialized_end=815
  _globals['_LISTWORKFLOWSREQUEST']._serialized_start=817
  _globals['_LISTWORKFLOWSREQUEST']._serialized_end=839
  _globals['_SCHEDULEWORKFLOWREQUEST']._serialized_start=841
  _globals['_SCHEDULEWORKFLOWREQUEST']._serialized_end=942
  _globals['_WORKFLOWVERSION']._serialized_start=945
  _globals['_WORKFLOWVERSION']._serialized_end=1123
  _globals['_WORKFLOWTRIGGEREVENTREF']._serialized_start=1125
  _globals['_WORKFLOWTRIGGEREVENTREF']._serialized_end=1188
  _globals['_WORKFLOWTRIGGERCRONREF']._serialized_start=1190
  _globals['_WORKFLOWTRIGGERCRONREF']._serialized_end=1247
  _globals['_TRIGGERWORKFLOWREQUEST']._serialized_start=1249
  _globals['_TRIGGERWORKFLOWREQUEST']._serialized_end=1302
  _globals['_TRIGGERWORKFLOWRESPONSE']._serialized_start=1304
  _globals['_TRIGGERWORKFLOWRESPONSE']._serialized_end=1354
  _globals['_WORKFLOWSERVICE']._serialized_start=1467
  _globals['_WORKFLOWSERVICE']._serialized_end=1672
# @@protoc_insertion_point(module_scope)
