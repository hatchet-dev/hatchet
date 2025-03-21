import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';
import { EventClient } from '@clients/event/event-client';
import { DispatcherClient } from '@clients/dispatcher/dispatcher-client';
import { AdminClient } from '@clients/admin/admin-client';
import {
  CallOptions,
  ChannelCredentials,
  ClientMiddlewareCall,
  createChannel,
  createClientFactory,
  Metadata,
} from 'nice-grpc';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { V0Worker, WorkerOpts } from '@clients/worker';
import { AxiosRequestConfig } from 'axios';
import { Logger } from '@util/logger';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { WorkflowDeclaration as V1Workflow } from '@hatchet/v1/workflow';
import { toV0Workflow } from '@hatchet/v1/client/worker';
import { ClientConfig, ClientConfigSchema } from './client-config';
import { ListenerClient } from '../listener/listener-client';
import { Api } from '../rest/generated/Api';
import api from '../rest';
import { CronClient } from './features/cron-client';
import { ScheduleClient } from './features/schedule-client';

export interface HatchetClientOptions {
  config_path?: string;
  credentials?: ChannelCredentials;
}

export const channelFactory = (config: ClientConfig, credentials: ChannelCredentials) =>
  createChannel(config.host_port, credentials, {
    'grpc.ssl_target_name_override': config.tls_config.server_name,
    'grpc.keepalive_timeout_ms': 60 * 1000,
    'grpc.client_idle_timeout_ms': 60 * 1000,
    // Send keepalive pings every 10 seconds, default is 2 hours.
    'grpc.keepalive_time_ms': 10 * 1000,
    // Allow keepalive pings when there are no gRPC calls.
    'grpc.keepalive_permit_without_calls': 1,
  });

export const addTokenMiddleware = (token: string) =>
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

export class InternalHatchetClient {
  config: ClientConfig;
  credentials: ChannelCredentials;
  event: EventClient;
  dispatcher: DispatcherClient;
  admin: AdminClient;
  api: Api;
  listener: ListenerClient;
  tenantId: string;

  logger: Logger;

  cron: CronClient;
  schedule: ScheduleClient;
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
      clientFactory
    );
    this.dispatcher = new DispatcherClient(
      this.config,
      channelFactory(this.config, this.credentials),
      clientFactory
    );
    this.listener = new ListenerClient(
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
      this.listener
    );

    this.logger = this.config.logger('HatchetClient', this.config.log_level);
    this.logger.debug(`Initialized HatchetClient`);

    // Feature Clients
    this.cron = new CronClient(this.tenantId, this.config, this.api, this.admin);
    this.schedule = new ScheduleClient(this.tenantId, this.config, this.api, this.admin);
  }

  static init(
    config?: Partial<ClientConfig>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): InternalHatchetClient {
    return new InternalHatchetClient(config, options, axiosConfig);
  }

  // @deprecated
  async run(workflow: string | V0Workflow): Promise<V0Worker> {
    this.logger.warn(
      'HatchetClient.run is deprecated and will be removed in a future release. Use HatchetClient.worker and Worker.start instead.'
    );
    const worker = await this.worker(workflow);
    worker.start();
    return worker;
  }

  async worker(
    workflow: string | V0Workflow,
    opts?: Omit<WorkerOpts, 'name'> | number
  ): Promise<V0Worker> {
    const name = typeof workflow === 'string' ? workflow : workflow.id;

    let options: WorkerOpts = {
      name,
    };

    if (typeof opts === 'number') {
      this.logger.warn(
        '@deprecated maxRuns param is deprecated and will be removed in a future release in favor of WorkerOpts'
      );
      options = { ...options, maxRuns: opts };
    } else {
      options = { ...options, ...opts };
    }

    const worker = new V0Worker(this, options);

    if (typeof workflow !== 'string') {
      await worker.registerWorkflow(workflow);
      return worker;
    }

    return worker;
  }

  webhooks(workflows: Array<V1Workflow<any, any> | V0Workflow>) {
    const worker = new V0Worker(this, {
      name: 'webhook-worker',
    });

    return worker.getHandler(workflows.map(toV0Workflow));
  }
}
