import type { EventsRpc } from '@hatchet/clients/event/rpc';
import type { PutStreamEventResponse } from '@hatchet/protoc/events/events';

/** Publishes stream chunks from a task run, the way `ctx.putStream` does from a worker. */
export class StreamsClient {
  constructor(private readonly rpc: EventsRpc) {}

  /**
   * Publishes one chunk on a task run's stream. `index` orders chunks that are published
   * concurrently. Resolves once the engine has the chunk.
   */
  put(
    taskRunExternalId: string,
    data: string | Uint8Array,
    index?: number
  ): Promise<PutStreamEventResponse> {
    const message = typeof data === 'string' ? new TextEncoder().encode(data) : data;
    return this.rpc.putStreamEvent({
      taskRunExternalId,
      createdAt: new Date(),
      message,
      eventIndex: index,
    });
  }
}
