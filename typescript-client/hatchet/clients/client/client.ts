import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';
import { EventClient } from '@clients/event/event-client';
import { DispatcherClient } from '@clients/dispatcher/dispatcher-client';
import { AdminClient } from '@clients/admin/admin-client';
import { ClientConfig, ClientConfigSchema } from './client-config';

export interface ClientOptions {
  config_path?: string;
}

export class Client {
  config: ClientConfig;
  event: EventClient;
  dispatcher: DispatcherClient;
  admin: AdminClient;

  constructor(config?: Partial<ClientConfig>, options?: ClientOptions) {
    // Initializes a new Client instance.
    // Loads config in teh following order: config param > yaml file > env vars

    const loaded = ConfigLoader.load_client_config({
      path: options?.config_path,
    });

    try {
      const valid = ClientConfigSchema.parse({ ...loaded, ...config });
      this.config = valid;
    } catch (e) {
      if (e instanceof z.ZodError) {
        throw new Error(`Invalid client config: ${e.message}`);
      }
      throw e;
    }

    this.event = new EventClient(this.config);
    this.dispatcher = new DispatcherClient(this.config);
    this.admin = new AdminClient(this.config);
  }

  static with_host_port(
    host: string,
    port: number,
    config?: Partial<ClientConfig>,
    options?: ClientOptions
  ): Client {
    return new Client(
      {
        ...config,
        host_port: `${host}:${port}`,
      },
      options
    );
  }
}
