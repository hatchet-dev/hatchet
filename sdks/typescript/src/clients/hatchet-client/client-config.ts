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

export type PreHookFn = (input: any, ctx: any) => Record<string, any> | void | Promise<Record<string, any> | void>;
export type PostHookFn = (output: any, ctx: any, input: any) => Record<string, any> | void | Promise<Record<string, any> | void>;

/**
 * Middleware hooks that run before/after every task invocation.
 *
 * Each hook can be a single function or an array of functions.
 * When an array is provided the functions run in order and each
 * result is merged into the value (input for `pre`, output for `post`).
 *
 * Each function returns only the **extra fields** to merge.
 * Return `void` (or `undefined`) from a hook to skip merging.
 */
export type TaskMiddleware = {
  pre?: PreHookFn | readonly PreHookFn[];
  post?: PostHookFn | readonly PostHookFn[];
};

type NonVoidReturn<F> = F extends (...args: any[]) => infer R
  ? Exclude<Awaited<R>, void | undefined>
  : {};

type MergeReturns<T> = T extends readonly [infer F, ...infer Rest]
  ? NonVoidReturn<F> & MergeReturns<Rest>
  : {};

export type InferMiddlewarePre<M> = M extends { pre: infer P }
  ? P extends (...args: any[]) => any
    ? NonVoidReturn<P>
    : P extends readonly any[]
      ? MergeReturns<P>
      : {}
  : {};

export type InferMiddlewarePost<M> = M extends { post: infer P }
  ? P extends (...args: any[]) => any
    ? NonVoidReturn<P>
    : P extends readonly any[]
      ? MergeReturns<P>
      : {}
  : {};

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
} & {
  logger: LogConstructor;
  middleware?: TaskMiddleware;
};
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
