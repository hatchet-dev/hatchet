import { t } from './copy';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { appRoutes } from '@/router';

// the key the serverless operator puts on the log line it writes when it delivers a task
const ENDPOINT_ID_KEY = 'hatchet__serverless_endpoint_id';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** A link to the serverless endpoint a task log line says the task was delivered to. */
export function endpointLinkFromLogMetadata(
  metadata: Record<string, unknown> | undefined,
  tenantId: string | undefined,
): LogLine['linkTo'] {
  const endpointId = metadata?.[ENDPOINT_ID_KEY];

  // tasks can write any metadata into their own logs, so only an id shaped like one links
  if (!tenantId || typeof endpointId !== 'string' || !UUID.test(endpointId)) {
    return undefined;
  }

  return {
    destination: appRoutes.tenantEndpointRoute.to,
    params: { tenant: tenantId, endpoint: endpointId },
    hoverText: t('taskLog.viewEndpoint'),
  };
}
