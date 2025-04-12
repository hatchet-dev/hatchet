import { ChannelCredentials } from 'nice-grpc';
import { z } from 'zod';
import { Logger, LogLevel } from '@util/logger';

const ClientTLSConfigSchema = z.object({
  tls_strategy: z.enum(['tls', 'mtls', 'none']).optional(),
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
  namespace: z.string().optional(),
  grpc_max_send_message_length: z.number().optional(),
  grpc_max_recv_message_length: z.number().optional(),
});

export type LogConstructor = (context: string, logLevel?: LogLevel) => Logger;

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
} & { logger: LogConstructor };
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
