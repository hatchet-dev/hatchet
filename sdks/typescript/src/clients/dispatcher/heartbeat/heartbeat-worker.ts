import { parentPort, workerData } from 'worker_threads';
import { Logger } from '@util/logger';
import {
  ClientConfig,
  HatchetLogger,
  addTokenMiddleware,
  channelFactory,
} from '@hatchet/clients/hatchet-client';
import { DispatcherClient as PbDispatcherClient } from '@hatchet/protoc/dispatcher';
import { ConfigLoader } from '@hatchet/util/config-loader';
import { Status, createClientFactory } from 'nice-grpc';
import { DispatcherClient } from '../dispatcher-client';

const HEARTBEAT_INTERVAL = 4000;

class HeartbeatWorker {
  heartbeatInterval: any;
  logger: Logger;
  client: PbDispatcherClient;
  workerId: string;
  timeLastHeartbeat = new Date().getTime();

  constructor(config: ClientConfig, workerId: string) {
    this.workerId = workerId;

    this.logger = new HatchetLogger(`Heartbeat`, config.log_level);

    const credentials = ConfigLoader.createCredentials(config.tls_config);
    const clientFactory = createClientFactory().use(addTokenMiddleware(config.token));

    const dispatcher = new DispatcherClient(
      { ...config, logger: (ctx, level) => new HatchetLogger(ctx, level) },
      channelFactory(config, credentials),
      clientFactory
    );

    this.client = dispatcher.client;
  }

  async start() {
    if (this.heartbeatInterval) {
      return;
    }

    const beat = async () => {
      try {
        this.logger.debug('Heartbeat sending...');
        await this.client.heartbeat({
          workerId: this.workerId,
          heartbeatAt: new Date(),
        });
        const now = new Date().getTime();

        const actualInterval = now - this.timeLastHeartbeat;

        if (actualInterval > HEARTBEAT_INTERVAL * 1.2) {
          this.logger.warn(
            `Heartbeat interval delay (${actualInterval}ms >> ${HEARTBEAT_INTERVAL}ms)`
          );
        }

        this.logger.debug(`Heartbeat sent ${actualInterval}ms ago`);
        this.timeLastHeartbeat = now;
      } catch (e: any) {
        if (e.code === Status.UNIMPLEMENTED) {
          // break out of interval
          this.logger.error('Heartbeat not implemented, closing heartbeat');
          this.stop();
          return;
        }

        this.logger.error(`Failed to send heartbeat: ${e.message}`);
      }
    };

    // start with a heartbeat
    await beat();
    this.heartbeatInterval = setInterval(beat, HEARTBEAT_INTERVAL);
  }

  stop() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
}

const heartbeat = new HeartbeatWorker(workerData.config, workerData.workerId);
heartbeat.start();

parentPort?.on('stop', () => {
  heartbeat.stop();
});
