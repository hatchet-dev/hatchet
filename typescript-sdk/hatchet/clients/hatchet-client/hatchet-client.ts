import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';
import { EventClient } from '@clients/event/event-client';
import { DispatcherClient } from '@clients/dispatcher/dispatcher-client';
import { AdminClient } from '@clients/admin/admin-client';
import { Channel, ChannelCredentials, createChannel } from 'nice-grpc';
import { Workflow } from '@hatchet/workflow';
import { Worker } from '@clients/worker';
import Logger from '@hatchet/util/logger/logger';
import { ClientConfig, ClientConfigSchema } from './client-config';

export interface HatchetClientOptions {
  config_path?: string;
  credentials?: ChannelCredentials;
}

export class HatchetClient {
  config: ClientConfig;
  credentials: ChannelCredentials;
  channel: Channel;

  event: EventClient;
  dispatcher: DispatcherClient;
  admin: AdminClient;

  logger: Logger;

  constructor(config?: Partial<ClientConfig>, options?: HatchetClientOptions) {
    // Initializes a new Client instance.
    // Loads config in the following order: config param > yaml file > env vars

    const loaded = ConfigLoader.load_client_config({
      path: options?.config_path,
    });

    try {
      const valid = ClientConfigSchema.parse({
        ...loaded,
        ...{ ...config, tls_config: { ...loaded.tls_config, ...config?.tls_config } },
      });
      this.config = valid;
    } catch (e) {
      if (e instanceof z.ZodError) {
        throw new Error(`Invalid client config: ${e.message}`);
      }
      throw e;
    }

    this.credentials =
      options?.credentials ?? ConfigLoader.createCredentials(this.config.tls_config);
    this.channel = createChannel(this.config.host_port, this.credentials, {
      'grpc.ssl_target_name_override': this.config.tls_config.server_name,
    });

    this.event = new EventClient(this.config, this.channel);
    this.dispatcher = new DispatcherClient(this.config, this.channel);
    this.admin = new AdminClient(this.config, this.channel);

    this.logger = new Logger('HatchetClient', this.config.log_level);

    this.logger.info(`Initialized HatchetClient`);
  }

  static with_host_port(
    host: string,
    port: number,
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions
  ): HatchetClient {
    return new HatchetClient(
      {
        ...config,
        host_port: `${host}:${port}`,
      },
      options
    );
  }

  static init(config?: Partial<ClientConfig>, options?: HatchetClientOptions): HatchetClient {
    return new HatchetClient(config, options);
  }

  async run(workflow: string | Workflow): Promise<Worker> {
    const worker = await this.worker(workflow);
    worker.start();
    return worker;
  }

  async worker(workflow: string | Workflow): Promise<Worker> {
    const name = typeof workflow === 'string' ? workflow : workflow.id;
    const worker = new Worker(this, {
      name,
    });

    if (typeof workflow !== 'string') {
      await worker.register_workflow(workflow);
      return worker;
    }

    return worker;
  }
}
