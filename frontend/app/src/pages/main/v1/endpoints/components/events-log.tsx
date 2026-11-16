import { escapeControls } from '../lib/escape';
import { t } from './copy';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  V1LogLineLevel,
  V1ServerlessEndpointEvent,
  V1ServerlessEndpointEventKind,
} from '@/lib/api';
import { appRoutes } from '@/router';
import { useMemo } from 'react';

// a single failed healthcheck is a warning: the endpoint turns unhealthy after 3 in a row
const LEVEL: Record<V1ServerlessEndpointEventKind, V1LogLineLevel> = {
  [V1ServerlessEndpointEventKind.HEALTHCHECK_FAILED]: V1LogLineLevel.WARN,
  [V1ServerlessEndpointEventKind.UNHEALTHY]: V1LogLineLevel.ERROR,
  [V1ServerlessEndpointEventKind.HEALTHY]: V1LogLineLevel.INFO,
  [V1ServerlessEndpointEventKind.CATALOG_REGISTERED]: V1LogLineLevel.INFO,
  [V1ServerlessEndpointEventKind.CATALOG_REJECTED]: V1LogLineLevel.ERROR,
  [V1ServerlessEndpointEventKind.DELIVERY_FAILED]: V1LogLineLevel.ERROR,
};

const toLogLine = (
  event: V1ServerlessEndpointEvent,
  tenantId: string,
): LogLine => {
  // the response body opens on its own; the rest of the details show as fields
  const { responseExcerpt, ...details } = event.details ?? {};
  const fields = Object.fromEntries(
    Object.entries(details).filter(
      ([, v]) =>
        v !== undefined && v !== null && !(Array.isArray(v) && !v.length),
    ),
  );

  const parts = [t(`event.${event.kind}`)];

  // the message is often the endpoint's own text; escaped so it cannot erase the label
  if (event.message) {
    parts.push(escapeControls(event.message));
  }

  if (details.suppressed) {
    parts.push(t('event.suppressed', { count: details.suppressed }));
  }

  return {
    timestamp: event.createdAt,
    level: LEVEL[event.kind],
    line: parts.join(' · '),
    metadata: fields,
    error: responseExcerpt ? escapeControls(responseExcerpt) : undefined,
    linkTo: details.taskRunExternalId
      ? {
          destination: appRoutes.tenantRunRoute.to,
          params: { tenant: tenantId, run: details.taskRunExternalId },
          hoverText: t('event.viewRun'),
        }
      : undefined,
  };
};

/** The operator's events for one endpoint, in the same view as task logs. */
export const EventsLog = ({
  events,
  isLoading,
}: {
  events: V1ServerlessEndpointEvent[];
  isLoading?: boolean;
}) => {
  const { tenantId } = useCurrentTenantId();
  const logs = useMemo(
    () => events.map((e) => toLogLine(e, tenantId)),
    [events, tenantId],
  );

  return (
    <div className="flex h-[480px] flex-col">
      <LogViewer
        logs={logs}
        isLoading={isLoading}
        showAttempt={false}
        emptyMessage={t('detail.events.empty')}
      />
    </div>
  );
};
