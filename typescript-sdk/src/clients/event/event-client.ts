import { Channel, ClientFactory } from 'nice-grpc';
import {
  EventsServiceClient,
  EventsServiceDefinition,
  PushEventRequest,
} from '@hatchet/protoc/events/events';
import HatchetError from '@util/errors/hatchet-error';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { Logger } from '@hatchet/util/logger';
import { retrier } from '@hatchet/util/retrier';

// eslint-disable-next-line no-shadow
export enum LogLevel {
  INFO = 'INFO',
  WARN = 'WARN',
  ERROR = 'ERROR',
  DEBUG = 'DEBUG',
}

export class EventClient {
  config: ClientConfig;
  client: EventsServiceClient;

  logger: Logger;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory) {
    this.config = config;
    this.client = factory.create(EventsServiceDefinition, channel);
    this.logger = new Logger(`Dispatcher`, config.log_level);
  }

  push<T>(type: string, input: T) {
    const req: PushEventRequest = {
      key: type,
      payload: JSON.stringify(input),
      eventTimestamp: new Date(),
    };

    try {
      const e = this.client.push(req);
      this.logger.info(`Event pushed: ${type}`);
      return e;
    } catch (e: any) {
      throw new HatchetError(e.message);
    }
  }

  putLog(stepRunId: string, log: string, level?: LogLevel) {
    const createdAt = new Date();

    try {
      retrier(
        async () =>
          this.client.putLog({
            stepRunId,
            createdAt,
            message: log,
            level: level || LogLevel.INFO,
          }),
        this.logger
      );
    } catch (e: any) {
      // log a warning, but this is not a fatal error
      this.logger.warn(`Could not put log: ${e.message}`);
    }
  }
}
