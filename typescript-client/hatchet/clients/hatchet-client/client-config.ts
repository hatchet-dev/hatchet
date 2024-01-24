import { ChannelCredentials } from 'nice-grpc';
import { z } from 'zod';

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

export type ClientConfig = z.infer<typeof ClientConfigSchema> & {
  credentials?: ChannelCredentials;
};
export type ClientTLSConfig = z.infer<typeof ClientTLSConfigSchema>;
