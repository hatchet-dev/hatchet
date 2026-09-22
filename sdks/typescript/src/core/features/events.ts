import { toHatchetError } from '@util/errors/hatchet-error';
import { retrier, type RetrierConfig } from '@hatchet/util/retrier';
import type { Logger } from '@hatchet/util/logger/logger';
import type { Event, Events } from '@hatchet/protoc/events/events';
import {
  buildBulkPushEventRequest,
  buildPushEventRequest,
  type EventsRpc,
} from '@hatchet/clients/event/rpc';
import type { EventWithMetadata, PushEventOptions } from '../types';

export interface EventsClientDeps {
  rpc: EventsRpc;
  logger: Logger;
  namespace?: string;
  retrier?: RetrierConfig;
}

/** Pushes events that trigger event-bound workflows. */
export class EventsClient {
  constructor(private readonly deps: EventsClientDeps) {}

  /**
   * Pushes one event. The key is namespaced the way the Node client namespaces it.
   * @returns the stored event
   */
  async push<T>(type: string, input: T, options: PushEventOptions = {}): Promise<Event> {
    const request = buildPushEventRequest(type, input, options, this.deps.namespace);
    try {
      const event = await retrier(
        () => this.deps.rpc.push(request),
        this.deps.logger,
        this.deps.retrier
      );
      this.deps.logger.debug(`Event pushed: ${request.key}`);
      return event;
    } catch (e: unknown) {
      throw toHatchetError(e);
    }
  }

  /**
   * Pushes many events of one type in a single call.
   * @returns the stored events
   */
  async bulkPush<T>(
    type: string,
    inputs: EventWithMetadata<T>[],
    options: PushEventOptions = {}
  ): Promise<Events> {
    const request = buildBulkPushEventRequest(type, inputs, options, this.deps.namespace);
    try {
      const events = await retrier(
        () => this.deps.rpc.bulkPush(request),
        this.deps.logger,
        this.deps.retrier
      );
      this.deps.logger.debug(`Bulk events pushed for type: ${request.events[0]?.key ?? type}`);
      return events;
    } catch (e: unknown) {
      throw toHatchetError(e);
    }
  }
}
