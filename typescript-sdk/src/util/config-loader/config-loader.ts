import { parse } from 'yaml';
import { readFileSync } from 'fs';
import * as p from 'path';
import { z } from 'zod';
import { ClientConfig, ClientConfigSchema } from '@clients/hatchet-client';
import { ChannelCredentials } from 'nice-grpc';
import { LogLevel } from '../logger/logger';
import { getAddressesFromJWT, getTenantIdFromJWT } from './token';

type EnvVars =
  | 'HATCHET_CLIENT_TOKEN'
  | 'HATCHET_CLIENT_TLS_STRATEGY'
  | 'HATCHET_CLIENT_HOST_PORT'
  | 'HATCHET_CLIENT_API_URL'
  | 'HATCHET_CLIENT_TLS_CERT_FILE'
  | 'HATCHET_CLIENT_TLS_KEY_FILE'
  | 'HATCHET_CLIENT_TLS_ROOT_CA_FILE'
  | 'HATCHET_CLIENT_TLS_SERVER_NAME'
  | 'HATCHET_CLIENT_LOG_LEVEL';

type TLSStrategy = 'tls' | 'mtls';

interface LoadClientConfigOptions {
  path?: string;
}

const DEFAULT_CONFIG_FILE = '.hatchet.yaml';

export class ConfigLoader {
  static loadClientConfig(config?: LoadClientConfigOptions): Partial<ClientConfig> {
    const yaml = this.loadYamlConfig(config?.path);
    const tlsConfig = {
      tls_strategy:
        yaml?.tls_config?.tls_strategy ??
        (this.env('HATCHET_CLIENT_TLS_STRATEGY') as TLSStrategy | undefined) ??
        'tls',
      cert_file: yaml?.tls_config?.cert_file ?? this.env('HATCHET_CLIENT_TLS_CERT_FILE')!,
      key_file: yaml?.tls_config?.key_file ?? this.env('HATCHET_CLIENT_TLS_KEY_FILE')!,
      ca_file: yaml?.tls_config?.ca_file ?? this.env('HATCHET_CLIENT_TLS_ROOT_CA_FILE')!,
      server_name: yaml?.tls_config?.server_name ?? this.env('HATCHET_CLIENT_TLS_SERVER_NAME')!,
    };

    const token = yaml?.token ?? this.env('HATCHET_CLIENT_TOKEN');
    let grpcBroadcastAddress: string | undefined;
    let apiUrl: string | undefined;
    const tenantId = getTenantIdFromJWT(token!);

    if (!tenantId) {
      throw new Error('Tenant ID not found in subject claim of token');
    }

    try {
      const addresses = getAddressesFromJWT(token!);

      grpcBroadcastAddress =
        yaml?.host_port ?? this.env('HATCHET_CLIENT_HOST_PORT') ?? addresses.grpcBroadcastAddress;

      apiUrl = yaml?.api_url ?? this.env('HATCHET_CLIENT_API_URL') ?? addresses.serverUrl;
    } catch (e) {
      grpcBroadcastAddress = yaml?.host_port ?? this.env('HATCHET_CLIENT_HOST_PORT');
      apiUrl = yaml?.api_url ?? this.env('HATCHET_CLIENT_API_URL');
    }

    return {
      token: yaml?.token ?? this.env('HATCHET_CLIENT_TOKEN'),
      host_port: grpcBroadcastAddress,
      api_url: apiUrl,
      tls_config: tlsConfig,
      log_level: yaml?.log_level ?? (this.env('HATCHET_CLIENT_LOG_LEVEL') as LogLevel) ?? 'INFO',
      tenant_id: tenantId,
    };
  }

  static get default_yaml_config_path() {
    return p.join(process.cwd(), DEFAULT_CONFIG_FILE);
  }

  static createCredentials(config: ClientConfig['tls_config']): ChannelCredentials {
    if (config.tls_strategy === 'tls') {
      const rootCerts = config.ca_file ? readFileSync(config.ca_file) : undefined;
      return ChannelCredentials.createSsl(rootCerts);
    }

    const rootCerts = config.ca_file ? readFileSync(config.ca_file) : null;
    const privateKey = config.key_file ? readFileSync(config.key_file) : null;
    const certChain = config.cert_file ? readFileSync(config.cert_file) : null;
    return ChannelCredentials.createSsl(rootCerts, privateKey, certChain);
  }

  static loadYamlConfig(path?: string): ClientConfig | undefined {
    try {
      const configFile = readFileSync(
        p.join(__dirname, path ?? this.default_yaml_config_path),
        'utf8'
      );

      const config = parse(configFile);

      ClientConfigSchema.partial().parse(config);

      return config as ClientConfig;
    } catch (e) {
      if (!path) return undefined;

      if (e instanceof z.ZodError) {
        throw new Error(`Invalid yaml config: ${e.message}`);
      }

      throw e;
    }
  }

  private static env(name: EnvVars): string | undefined {
    return process.env[name];
  }
}
