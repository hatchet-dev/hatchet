export type { OpenTelemetryConfig } from '@hatchet/clients/hatchet-client/client-config';

export const DEFAULT_CONFIG = {
  excludedAttributes: [] as string[],
  includeTaskNameInSpanName: false,
};
