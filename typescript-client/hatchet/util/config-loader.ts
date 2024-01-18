import { ClientConfig } from '../models/client';

type EnvVars =
  | 'HATCHET_CLIENT_TENANT_ID'
  | 'HATCHET_CLIENT_HOST_PORT'
  | 'HATCHET_CLIENT_TLS_CERT_FILE'
  | 'HATCHET_CLIENT_TLS_KEY_FILE'
  | 'HATCHET_CLIENT_TLS_ROOT_CA_FILE'
  | 'HATCHET_CLIENT_TLS_SERVER_NAME';

export default class ConfigLoader {
  static load_client_config(): ClientConfig {
    return {
      tenant_id: this.env('HATCHET_CLIENT_TENANT_ID'),
      host_port: this.env('HATCHET_CLIENT_HOST_PORT'),
      tls_config: {
        cert_file: this.env('HATCHET_CLIENT_TLS_CERT_FILE')!,
        key_file: this.env('HATCHET_CLIENT_TLS_KEY_FILE')!,
        ca_file: this.env('HATCHET_CLIENT_TLS_ROOT_CA_FILE')!,
        server_name: this.env('HATCHET_CLIENT_TLS_SERVER_NAME')!,
      },
    };
  }

  private static env(name: EnvVars): string | undefined {
    return process.env[name];
  }
}
