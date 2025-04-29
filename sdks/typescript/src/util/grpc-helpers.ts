import { ClientConfig } from '@hatchet/clients/hatchet-client';
import {
  ChannelCredentials,
  CompatServiceDefinition,
  createChannel,
  createClientFactory,
} from 'nice-grpc';
import { ClientMiddlewareCall, CallOptions, Metadata } from 'nice-grpc-common';
import { ConfigLoader } from './config-loader';

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

export const createGrpcClient = <T extends CompatServiceDefinition>(
  config: ClientConfig,
  serviceDefinition: T
) => {
  const credentials = ConfigLoader.createCredentials(config.tls_config);
  const clientFactory = createClientFactory().use(addTokenMiddleware(config.token));
  const channel = channelFactory(config, credentials);
  return {
    factory: clientFactory,
    channel,
    client: clientFactory.create<T>(serviceDefinition, channel),
  };
};
