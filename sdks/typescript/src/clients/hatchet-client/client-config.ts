import { ChannelCredentials } from 'nice-grpc';
import { z } from 'zod';
import { Logger, LogLevel } from '@util/logger';

// Cancellation timings are specified in integer milliseconds.
const DurationMsSchema = z.number().int().nonnegative().finite();

const ClientTLSConfigSchema = z.object({
  tls_strategy: z.enum(['tls', 'mtls', 'none']).optional(),
  cert_file: z.string().optional(),
  ca_file: z.string().optional(),
  key_file: z.string().optional(),
  server_name: z.string().optional(),
});

const HealthcheckConfigSchema = z.object({
  enabled: z.boolean().optional().default(false),
  port: z.number().optional().default(8001),
});

export const ClientConfigSchema = z.object({
  token: z.string(),
  tls_config: ClientTLSConfigSchema,
  healthcheck: HealthcheckConfigSchema.optional(),
  host_port: z.string(),
  api_url: z.string(),
  log_level: z.enum(['OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR']).optional(),
  tenant_id: z.string(),
  namespace: z.string().optional(),
  cancellation_grace_period: DurationMsSchema.optional().default(1000),
  cancellation_warning_threshold: DurationMsSchema.optional().default(300),
});

export type LogConstructor = (context: string, logLevel?: LogLevel) => Logger;

type ClientConfigInferred = z.infer<typeof ClientConfigSchema>;

// Backwards-compatible: allow callers to omit these (schema supplies defaults when parsed).
type ClientConfigCancellationCompat = {
  cancellation_grace_period?: ClientConfigInferred['cancellation_grace_period'];
  cancellation_warning_threshold?: ClientConfigInferred['cancellation_warning_threshold'];
};

export type ClientConfig = Omit<
  ClientConfigInferred,
  'cancellation_grace_period' | 'cancellation_warning_threshold'
> &
  ClientConfigCancellationCompat & {
    credentials?: ChannelCredentials;
  } & { logger: LogConstructor };
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
