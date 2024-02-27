import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';
import { EventClient } from '@clients/event/event-client';
import { DispatcherClient } from '@clients/dispatcher/dispatcher-client';
import { AdminClient } from '@clients/admin/admin-client';
import {
  CallOptions,
  Channel,
  ChannelCredentials,
  ClientMiddlewareCall,
  Metadata,
  createChannel,
  createClientFactory,
} from 'nice-grpc';
import { Workflow } from '@hatchet/workflow';
import { Worker } from '@clients/worker';
import Logger from '@hatchet/util/logger/logger';
import { AxiosRequestConfig } from 'axios';
import { ClientConfig, ClientConfigSchema } from './client-config';
import { ListenerClient } from '../listener/listener-client';
import { Api } from '../rest/generated/Api';
import api from '../rest';

export interface HatchetClientOptions {
  config_path?: string;
  credentials?: ChannelCredentials;
}

const addTokenMiddleware = (token: string) =>
  async function* _<Request, Response>(
    call: ClientMiddlewareCall<Request, Response>,
    options: CallOptions
  ) {
    const optionsWithAuth: CallOptions = {
      ...options,
      metadata: new Metadata({ authorization: `bearer ${token}` }),
    };

    if (!call.responseStream) {
      const response = yield* call.next(call.request, optionsWithAuth);

      return response;
    }

    for await (const response of call.next(call.request, optionsWithAuth)) {
      yield response;
    }

    return undefined;
  };

export class HatchetClient {
  config: ClientConfig;
  credentials: ChannelCredentials;
  channel: Channel;

  event: EventClient;
  dispatcher: DispatcherClient;
  admin: AdminClient;
  api: Api;
  listener: ListenerClient;
  tenantId: string;

  logger: Logger;

  constructor(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosOpts?: AxiosRequestConfig
  ) {
    // Initializes a new Client instance.
    // Loads config in the following order: config param > yaml file > env vars

    const loaded = ConfigLoader.loadClientConfig(config, {
      path: options?.config_path,
    });

    try {
      const valid = ClientConfigSchema.parse(loaded);
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
      'grpc.keepalive_timeout_ms': 60 * 1000,
      'grpc.client_idle_timeout_ms': 60 * 1000,
      // Send keepalive pings every 10 seconds, default is 2 hours.
      'grpc.keepalive_time_ms': 10 * 1000,
      // Allow keepalive pings when there are no gRPC calls.
      'grpc.keepalive_permit_without_calls': 1,
    });

    const clientFactory = createClientFactory().use(addTokenMiddleware(this.config.token));

    this.tenantId = this.config.tenant_id;
    this.api = api(this.config.api_url, this.config.token, axiosOpts);
    this.event = new EventClient(this.config, this.channel, clientFactory);
    this.dispatcher = new DispatcherClient(this.config, this.channel, clientFactory);
    this.admin = new AdminClient(this.config, this.channel, clientFactory, this.api, this.tenantId);
    this.listener = new ListenerClient(this.config, this.channel, clientFactory);

    this.logger = new Logger('HatchetClient', this.config.log_level);

    this.logger.info(`Initialized HatchetClient`);
  }

  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): HatchetClient {
    return new HatchetClient(config, options, axiosConfig);
  }

  // @deprecated
  async run(workflow: string | Workflow): Promise<Worker> {
    const worker = await this.worker(workflow);
    worker.start();
    return worker;
  }

  async worker(workflow: string | Workflow, maxRuns?: number): Promise<Worker> {
    const name = typeof workflow === 'string' ? workflow : workflow.id;
    const worker = new Worker(this, {
      name,
      maxRuns,
    });

    if (typeof workflow !== 'string') {
      await worker.registerWorkflow(workflow);
      return worker;
    }

    return worker;
  }
}
