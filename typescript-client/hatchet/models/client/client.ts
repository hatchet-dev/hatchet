import { z } from 'zod';
import { ConfigLoader } from '@util/config-loader';

const ClientTLSConfigSchema = z.object({
  cert_file: z.string(),
  ca_file: z.string(),
  key_file: z.string(),
  server_name: z.string(),
});

export const ClientConfigSchema = z.object({
  tenant_id: z.string(),
  tls_config: ClientTLSConfigSchema,
  host_port: z.string(),
});

export type ClientConfig = z.infer<typeof ClientConfigSchema>;
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;

interface ClientOptions {
  config_path?: string;
}

export class Client {
  config: ClientConfig;

  constructor(config: Partial<ClientConfig>, options?: ClientOptions) {
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

  admin: any; // TODO: AdminClient
  dispatcher: any; // TODO: DispatcherClient
  event: any; // TODO: EventClient
}
