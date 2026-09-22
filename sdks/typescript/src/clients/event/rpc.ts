import { Client, createClient } from '@connectrpc/connect';
import {
  BulkPushEventRequest,
  Event,
  Events,
  PushEventRequest,
  PutLogRequest,
  PutLogResponse,
  PutStreamEventRequest,
  PutStreamEventResponse,
} from '@hatchet/protoc/events/events';
import {
  BulkPushEventRequestSchema,
  EventSchema,
  EventsSchema,
  EventsService,
  PushEventRequestSchema,
  PutLogRequestSchema,
  PutLogResponseSchema,
  PutStreamEventRequestSchema,
  PutStreamEventResponseSchema,
} from '@hatchet/protoc-es/events/events_pb';
import { fromProtobufEs, toProtobufEs, type DeepPartial } from '@clients/transport/message-bridge';
import type { Transport } from '@clients/transport/transport';
import type { EventWithMetadata, PushEventOptions } from '@hatchet/core/types';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { parentRunContextManager } from '@hatchet/v1/parent-run-context-vars';

/** The levels a task run log line is stored with. */
export enum LogLevel {
  INFO = 'INFO',
  WARN = 'WARN',
  ERROR = 'ERROR',
  DEBUG = 'DEBUG',
}

/**
 * The events RPCs the SDK calls, typed with the SDK's message types and served by a Connect
 * client on the given transport.
 */
export interface EventsRpc {
  push(request: DeepPartial<PushEventRequest>): Promise<Event>;
  bulkPush(request: DeepPartial<BulkPushEventRequest>): Promise<Events>;
  putLog(request: DeepPartial<PutLogRequest>): Promise<PutLogResponse>;
  putStreamEvent(request: DeepPartial<PutStreamEventRequest>): Promise<PutStreamEventResponse>;
}

export function createEventsRpc(transport: Transport): EventsRpc {
  const client: Client<typeof EventsService> = createClient(EventsService, transport);

  return {
    push: async (request) =>
      fromProtobufEs(
        Event,
        EventSchema,
        await client.push(toProtobufEs(PushEventRequestSchema, PushEventRequest, request))
      ),
    bulkPush: async (request) =>
      fromProtobufEs(
        Events,
        EventsSchema,
        await client.bulkPush(
          toProtobufEs(BulkPushEventRequestSchema, BulkPushEventRequest, request)
        )
      ),
    putLog: async (request) =>
      fromProtobufEs(
        PutLogResponse,
        PutLogResponseSchema,
        await client.putLog(toProtobufEs(PutLogRequestSchema, PutLogRequest, request))
      ),
    putStreamEvent: async (request) =>
      fromProtobufEs(
        PutStreamEventResponse,
        PutStreamEventResponseSchema,
        await client.putStreamEvent(
          toProtobufEs(PutStreamEventRequestSchema, PutStreamEventRequest, request)
        )
      ),
  };
}

/**
 * When an event is pushed from inside a task, the run that pushed it is recorded in the
 * event's metadata so the dashboard can link the two.
 */
function injectSourceInfo(metadata: Record<string, string>): Record<string, string> {
  const ctx = parentRunContextManager.getContext();
  if (!ctx?.parentId || !ctx?.parentTaskRunExternalId) {
    return metadata;
  }
  return {
    ...metadata,
    hatchet__source_workflow_run_id: ctx.parentId,
    hatchet__source_step_run_id: ctx.parentTaskRunExternalId,
  };
}

/**
 * Builds the `Push` request for one event: the key is namespaced, the payload and metadata
 * are JSON, and the source run is recorded when there is one. The Node client and the core
 * client send exactly this.
 */
export function buildPushEventRequest<T>(
  type: string,
  input: T,
  options: PushEventOptions,
  namespace: string | undefined
): PushEventRequest {
  const enhancedMetadata = injectSourceInfo(options.additionalMetadata ?? {});

  return {
    key: applyNamespace(type, namespace),
    payload: JSON.stringify(input),
    eventTimestamp: new Date(),
    additionalMetadata:
      Object.keys(enhancedMetadata).length > 0 ? JSON.stringify(enhancedMetadata) : undefined,
    priority: options.priority,
    scope: options.scope,
  };
}

/**
 * Builds the `BulkPush` request for events of one type. Each event's own metadata, priority
 * and scope win over the shared options.
 */
export function buildBulkPushEventRequest<T>(
  type: string,
  inputs: EventWithMetadata<T>[],
  options: PushEventOptions,
  namespace: string | undefined
): BulkPushEventRequest {
  const namespacedType = applyNamespace(type, namespace);

  return {
    events: inputs.map((input) => {
      const baseMeta =
        (input.additionalMetadata as Record<string, string>) ?? options.additionalMetadata ?? {};
      const enhanced = injectSourceInfo(baseMeta);

      return {
        key: namespacedType,
        payload: JSON.stringify(input.payload),
        eventTimestamp: new Date(),
        additionalMetadata: Object.keys(enhanced).length > 0 ? JSON.stringify(enhanced) : undefined,
        priority: input.priority ?? options.priority,
        scope: input.scope ?? options.scope,
      };
    }),
  };
}
