import { ChannelCredentials } from 'nice-grpc';
import { z } from 'zod';

const ClientTLSConfigSchema = z.object({
  tls_strategy: z.enum(['tls', 'mtls']).optional(),
  cert_file: z.string().optional(),
  ca_file: z.string().optional(),
  key_file: z.string().optional(),
  server_name: z.string().optional(),
});

export const ClientConfigSchema = z.object({
  token: z.string(),
  tls_config: ClientTLSConfigSchema,
  host_port: z.string(),
  api_url: z.string(),
  log_level: z.enum(['OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR']).optional(),
  tenant_id: z.string(),
});

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
};
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
