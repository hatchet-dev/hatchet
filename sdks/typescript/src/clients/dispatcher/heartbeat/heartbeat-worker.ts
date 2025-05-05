import { parentPort, workerData } from 'worker_threads';
import { Logger } from '@util/logger';
import { ClientConfig, HatchetLogger } from '@hatchet/clients/hatchet-client';
import { DispatcherClient as PbDispatcherClient } from '@hatchet/protoc/dispatcher';
import { ConfigLoader } from '@hatchet/util/config-loader';
import { Status, createClientFactory } from 'nice-grpc';
import { addTokenMiddleware, channelFactory } from '@hatchet/util/grpc-helpers';
import { DispatcherClient } from '../dispatcher-client';
import { HeartbeatMessage, STOP_HEARTBEAT } from './heartbeat-controller';

const HEARTBEAT_INTERVAL = 4000;

const postMessage = (message: HeartbeatMessage) => {
  parentPort?.postMessage(message);
};

class HeartbeatWorker {
  heartbeatInterval: any;
  logger: Logger;
  client: PbDispatcherClient;
  workerId: string;
  timeLastHeartbeat = new Date().getTime();

  constructor(config: ClientConfig, workerId: string) {
    this.workerId = workerId;

    this.logger = new HatchetLogger(`HeartbeatThread`, config.log_level);

    this.logger.debug('Heartbeat thread starting...');
    const credentials = ConfigLoader.createCredentials(config.tls_config);
    const clientFactory = createClientFactory().use(addTokenMiddleware(config.token));

    const dispatcher = new DispatcherClient(
      { ...config, logger: (ctx, level) => new HatchetLogger(ctx, level) },
      channelFactory(config, credentials),
      clientFactory
    );

    this.client = dispatcher.client;
    postMessage({
      type: 'debug',
      message: 'Heartbeat thread started.',
    });
  }

  async start() {
    if (this.heartbeatInterval) {
      return;
    }

    const beat = async () => {
      try {
        this.logger.debug('Heartbeat sending...');
        postMessage({
          type: 'debug',
          message: 'Heartbeat sending...',
        });

        await this.client.heartbeat({
          workerId: this.workerId,
          heartbeatAt: new Date(),
        });
        const now = new Date().getTime();

        const actualInterval = now - this.timeLastHeartbeat;

        if (actualInterval > HEARTBEAT_INTERVAL * 1.2) {
          const message = `Heartbeat interval delay (${actualInterval}ms >> ${HEARTBEAT_INTERVAL}ms)`;
          this.logger.warn(message);
          postMessage({
            type: 'warn',
            message,
          });
        }

        this.logger.debug(`Heartbeat sent ${actualInterval}ms ago`);
        postMessage({
          type: 'debug',
          message: `Heartbeat sent ${actualInterval}ms ago`,
        });
        this.timeLastHeartbeat = now;
      } catch (e: any) {
        if (e.code === Status.UNIMPLEMENTED) {
          // break out of interval
          const message = 'Heartbeat not implemented, closing heartbeat';
          this.logger.debug(message);
          postMessage({
            type: 'error',
            message,
          });
          this.stop();
          return;
        }

        const message = `Failed to send heartbeat: ${e.message}`;
        this.logger.debug(message);
        postMessage({
          type: 'error',
          message,
        });
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

parentPort?.on(STOP_HEARTBEAT, () => {
  heartbeat.stop();
});
