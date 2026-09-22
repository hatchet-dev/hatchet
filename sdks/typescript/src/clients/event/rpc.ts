import {
  BulkPushEventRequest,
  EventsServiceClient,
  EventsServiceDefinition,
  PushEventRequest,
} from '@hatchet/protoc/events/events';
import { EventsService } from '@hatchet/protoc-es/events/events_pb';
import { createTsProtoClient } from '@clients/transport/ts-proto-client';
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
 * The `EventsService` client, every RPC of the generated `EventsServiceClient` interface,
 * served by a Connect client on the given transport. The Node and core clients both call
 * through it.
 */
export function createEventsRpc(transport: Transport): EventsServiceClient {
  return createTsProtoClient(EventsServiceDefinition, EventsService, transport);
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
