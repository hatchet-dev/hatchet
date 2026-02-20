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

const TaskMiddlewareSchema = z
  .object({
    pre: z.any().optional(),
    post: z.any().optional(),
  })
  .optional();

export const ClientConfigSchema = z.object({
  token: z.string(),
  tls_config: ClientTLSConfigSchema,
  healthcheck: HealthcheckConfigSchema.optional(),
  host_port: z.string(),
  api_url: z.string(),
  log_level: z.enum(['OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR']).optional(),
  tenant_id: z.string(),
  namespace: z.string().optional(),
  middleware: TaskMiddlewareSchema,
});

export type LogConstructor = (context: string, logLevel?: LogLevel) => Logger;

/**
 * Middleware hooks that run before/after every task invocation.
 *
 * `pre`  returns only the **extra fields** to merge into the task's input.
 * `post` returns only the **extra fields** to merge into the task's output.
 *
 * Return `void` (or `undefined`) from either hook to skip merging.
 */
export type TaskMiddleware<
  PreAdd extends Record<string, any> = {},
  PostAdd extends Record<string, any> = {},
> = {
  pre?: (input: any, ctx: any) => PreAdd | void | Promise<PreAdd | void>;
  post?: (output: any, ctx: any, input: any) => PostAdd | void | Promise<PostAdd | void>;
};

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
} & {
  logger: LogConstructor;
  middleware?: TaskMiddleware<any, any>;
};
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
