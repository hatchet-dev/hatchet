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

const HealthcheckConfigSchema = z.object({
  enabled: z.boolean().optional().default(false),
  port: z.number().optional().default(8001),
});

export const OpenTelemetryConfigSchema = z.object({
  /**
   * List of attribute keys to exclude from spans.
   * Useful for filtering sensitive or verbose data like payloads.
   */
  excludedAttributes: z.array(z.string()).optional().default([]),

  /**
   * If true, includes the task name in the span name for start_step_run spans.
   * e.g., "hatchet.start_step_run.my_task" instead of "hatchet.start_step_run"
   */
  includeTaskNameInSpanName: z.boolean().optional().default(false),
});

export type OpenTelemetryConfig = z.infer<typeof OpenTelemetryConfigSchema>;

export const ClientConfigSchema = z.object({
  token: z.string(),
  tls_config: ClientTLSConfigSchema,
  healthcheck: HealthcheckConfigSchema.optional(),
  host_port: z.string(),
  api_url: z.string(),
  log_level: z.enum(['OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR']).optional(),
  tenant_id: z.string(),
  namespace: z.string().optional(),
  otel: OpenTelemetryConfigSchema.optional(),
});

export type LogConstructor = (context: string, logLevel?: LogLevel) => Logger;

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
} & { logger: LogConstructor };
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
