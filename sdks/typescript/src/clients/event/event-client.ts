import { Channel, ClientFactory } from 'nice-grpc';
import { createNodeTransport, type Transport } from '@clients/transport';
import { getErrorMessage, toHatchetError } from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';
import { retrier } from '@hatchet/util/retrier';
import { applyNamespace } from '@hatchet/util/apply-namespace';
import { HatchetClient } from '@hatchet/v1';
import type { EventWithMetadata, PushEventOptions } from '@hatchet/core/types';
import {
  buildBulkPushEventRequest,
  buildPushEventRequest,
  createEventsRpc,
  LogLevel,
  type EventsRpc,
} from './rpc';

export type { EventsRpc, EventWithMetadata, PushEventOptions };
export { createEventsRpc, LogLevel };

export class EventClient {
  config: ClientConfig;
  client: EventsRpc;
  retrier: typeof retrier;
  api: HatchetClient['api'];
  tenantId: string;

  logger: Logger;

  // The channel and factory arguments are the streaming clients' and are unused here; they stay
  // in the signature so deep imports constructed with the original four arguments keep working.
  constructor(
    config: ClientConfig,
    _channel: Channel,
    _factory: ClientFactory,
    api: HatchetClient['api'],
    transport: Transport = createNodeTransport(config)
  ) {
    this.config = config;
    this.client = createEventsRpc(transport);
    this.logger = config.logger(`Dispatcher`, config.log_level);
    this.retrier = retrier;
    this.api = api;
    this.tenantId = config.tenant_id;
  }

  /**
   * @important This method is instrumented by HatchetInstrumentor._patchPushEvent.
   * Keep the signature in sync with the instrumentor wrapper.
   */
  push<T>(type: string, input: T, options: PushEventOptions = {}) {
    const namespacedType = applyNamespace(type, this.config.namespace);
    const req = buildPushEventRequest(type, input, options, this.config.namespace);

    return this.retrier(async () => this.client.push(req), this.logger, this.config.retrier)
      .then((result) => {
        this.logger.info(`Event pushed: ${namespacedType}`);
        return result;
      })
      .catch((e: unknown) => {
        throw toHatchetError(e);
      });
  }

  /**
   * @important This method is instrumented by HatchetInstrumentor._patchBulkPushEvent.
   * Keep the signature in sync with the instrumentor wrapper.
   */
  bulkPush<T>(type: string, inputs: EventWithMetadata<T>[], options: PushEventOptions = {}) {
    const namespacedType = applyNamespace(type, this.config.namespace);
    const req = buildBulkPushEventRequest(type, inputs, options, this.config.namespace);

    return this.retrier(async () => this.client.bulkPush(req), this.logger, this.config.retrier)
      .then((result) => {
        this.logger.info(`Bulk events pushed for type: ${namespacedType}`);
        return result;
      })
      .catch((e: unknown) => {
        throw toHatchetError(e);
      });
  }

  async putLog(
    taskRunExternalId: string,
    log: string,
    level?: LogLevel,
    taskRetryCount?: number,
    metadata?: Record<string, unknown>
  ) {
    const createdAt = new Date();

    if (log.length > 1_000) {
      this.logger.warn(`log is too long, skipping: ${log.length} characters`);
      return;
    }

    //  fire and forget the log
    await this.client
      .putLog({
        taskRunExternalId,
        createdAt,
        message: log,
        level: level || LogLevel.INFO,
        taskRetryCount,
        metadata: metadata ? JSON.stringify(metadata) : undefined,
      })
      .catch((e: unknown) => {
        this.logger.warn(`Could not put log: ${getErrorMessage(e)}`);
      });
  }

  async putStream(taskRunExternalId: string, data: string | Uint8Array, index: number | undefined) {
    const createdAt = new Date();

    let dataBytes: Uint8Array;
    if (typeof data === 'string') {
      dataBytes = new TextEncoder().encode(data);
    } else if (data instanceof Uint8Array) {
      dataBytes = data;
    } else {
      throw new Error('Invalid data type. Expected string or Uint8Array.');
    }

    retrier(
      async () =>
        this.client.putStreamEvent({
          taskRunExternalId,
          createdAt,
          message: dataBytes,
          eventIndex: index,
        }),
      this.logger
    ).catch((e: unknown) => {
      this.logger.warn(`Could not put log: ${getErrorMessage(e)}`);
    });
  }

  async list(opts?: Parameters<typeof this.api.v1EventList>[1]) {
    const { data } = await this.api.v1EventList(this.tenantId, opts);
    return data;
  }
}
