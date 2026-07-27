import { Logger } from '@hatchet/util/logger';
import { DispatcherClient as PbDispatcherClient } from '@hatchet/protoc/dispatcher';
import { Worker } from 'worker_threads';
import path from 'path';
import { runThreaded } from '@hatchet/util/thread-helper';
import { ClientConfig } from '../../hatchet-client';
import { DispatcherClient } from '../dispatcher-client';
import { z } from 'zod/v4';

const HeartbeatMessageSchema = z.object({
  type: z.enum(['info', 'warn', 'error', 'debug']),
  message: z.string(),
});

export type HeartbeatMessage = z.infer<typeof HeartbeatMessageSchema>;

export const isHeartbeatMessage = (message: unknown): message is HeartbeatMessage =>
  HeartbeatMessageSchema.safeParse(message).success;

export const STOP_HEARTBEAT = 'stop';
export class Heartbeat {
  config: ClientConfig;
  client: PbDispatcherClient;
  workerId: string;
  logger: Logger;

  heartbeatWorker: Worker | undefined;

  constructor(client: DispatcherClient, workerId: string) {
    this.config = client.config;
    this.client = client.client;
    this.workerId = workerId;
    this.logger = client.config.logger(`HeartbeatController`, this.config.log_level);
  }

  async start() {
    if (!this.heartbeatWorker) {
      const {
        middleware: _m,
        logger: _l,
        ...clonableConfig
      } = this.config as Record<string, unknown>;
      this.heartbeatWorker = runThreaded(path.join(__dirname, './heartbeat-worker'), {
        workerData: {
          config: clonableConfig,
          workerId: this.workerId,
        },
      });

      this.heartbeatWorker.on('message', (message: unknown) => {
        if (!isHeartbeatMessage(message)) {
          return;
        }

        this.logger[message.type](message.message);
      });
    }
  }

  async stop() {
    this.heartbeatWorker?.postMessage(STOP_HEARTBEAT);
    this.heartbeatWorker?.terminate();
    this.heartbeatWorker = undefined;
  }
}
