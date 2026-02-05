import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';
import { EventClient } from '@clients/event/event-client';
import { DispatcherClient } from '@clients/dispatcher/dispatcher-client';
import { AdminClient } from '@clients/admin/admin-client';
import { ChannelCredentials, createClientFactory } from 'nice-grpc';
import { AxiosRequestConfig } from 'axios';
import { Logger } from '@util/logger';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { RunsClient } from '@hatchet/v1';
import { addTokenMiddleware, channelFactory } from '@hatchet/util/grpc-helpers';
import { ClientConfig, ClientConfigSchema } from './client-config';
import { RunListenerClient } from '../listeners/run-listener/child-listener-client';
import { Api } from '../rest/generated/Api';
import api from '../rest';
import { DurableListenerClient } from '../listeners/durable-listener/durable-listener-client';

export interface HatchetClientOptions {
  config_path?: string;
  credentials?: ChannelCredentials;
}

export class LegacyHatchetClient {
  config: ClientConfig;
  credentials: ChannelCredentials;
  event: EventClient;
  dispatcher: DispatcherClient;
  admin: AdminClient;
  api: Api;
  runs: RunsClient | undefined;

  listener: RunListenerClient;
  tenantId: string;

  durableListener: DurableListenerClient;

  logger: Logger;

  constructor(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosOpts?: AxiosRequestConfig,
    runs?: RunsClient,
    listener?: RunListenerClient
  ) {
    // Initializes a new Client instance.
    // Loads config in the following order: config param > yaml file > env vars

    this.runs = runs;

    const loaded = ConfigLoader.loadClientConfig(config, {
      path: options?.config_path,
    });

    try {
      const valid = ClientConfigSchema.parse(loaded);

      let logConstructor = config?.logger;

      if (logConstructor == null) {
        logConstructor = DEFAULT_LOGGER;
      }

      this.config = {
        ...valid,
        logger: logConstructor,
      };
    } catch (e) {
      if (e instanceof z.ZodError) {
        throw new Error(`Invalid client config: ${e.message}`);
      }
      throw e;
    }

    this.credentials =
      options?.credentials ?? ConfigLoader.createCredentials(this.config.tls_config);

    const clientFactory = createClientFactory().use(addTokenMiddleware(this.config.token));

    this.tenantId = this.config.tenant_id;
    this.api = api(this.config.api_url, this.config.token, axiosOpts);
    this.event = new EventClient(
      this.config,
      channelFactory(this.config, this.credentials),
      clientFactory,
      this
    );
    this.dispatcher = new DispatcherClient(
      this.config,
      channelFactory(this.config, this.credentials),
      clientFactory
    );
    this.listener =
      listener ||
      new RunListenerClient(
        this.config,
        channelFactory(this.config, this.credentials),
        clientFactory,
        this.api
      );

    this.admin = new AdminClient(
      this.config,
      channelFactory(this.config, this.credentials),
      clientFactory,
      this.api,
      this.tenantId,
      this.listener,
      this.runs
    );

    this.durableListener = new DurableListenerClient(
      this.config,
      channelFactory(this.config, this.credentials),
      clientFactory,
      this.api
    );

    this.logger = this.config.logger('HatchetClient', this.config.log_level);
    this.logger.debug(`Initialized HatchetClient`);
  }

  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): LegacyHatchetClient {
    return new LegacyHatchetClient(config, options, axiosConfig);
  }
}
