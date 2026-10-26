import { Copy, t } from '../components/copy';
import { DiagnosisPanel } from '../components/diagnosis-panel';
import { EndpointActionsMenu } from '../components/endpoint-actions';
import { EndpointStatusBadge } from '../components/endpoint-status-badge';
import { EntitlementGate } from '../components/entitlement-gate';
import { EventsLog } from '../components/events-log';
import { PlatformBadge } from '../components/platform-name';
import {
  ENDPOINTS_REFETCH_INTERVAL_MS,
  useEndpointMutations,
} from '../hooks/use-endpoints';
import { diagnoseEndpoint } from '../lib/diagnose';
import RelativeDate from '@/components/v1/molecules/relative-date';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import { Loading } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import useCanWrite from '@/hooks/use-can-write';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { useApiError } from '@/lib/hooks';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { AxiosError, isAxiosError } from 'axios';
import { useMemo } from 'react';

const Row = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <div>
    <div className="text-xs text-gray-500 dark:text-gray-400">{label}</div>
    <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
      {children}
    </div>
  </div>
);

const UrlRow = ({ label, url }: { label: string; url: string }) => (
  <Row label={label}>
    <div className="flex flex-row items-center gap-1">
      <code className="min-w-0 break-all font-mono text-xs">{url}</code>
      <CopyToClipboard text={url} />
    </div>
  </Row>
);

// groups action ids ("workflow:task") by their workflow name
const groupActions = (actions: string[]) => {
  const groups = new Map<string, string[]>();

  for (const action of [...actions].sort()) {
    const sep = action.indexOf(':');
    const workflow = sep >= 0 ? action.slice(0, sep) : action;
    groups.set(workflow, [...(groups.get(workflow) ?? []), action]);
  }

  return [...groups.entries()];
};

export default function EndpointDetail() {
  return (
    <EntitlementGate>
      <EndpointDetailContent />
    </EntitlementGate>
  );
}

function EndpointDetailContent() {
  const { tenantId } = useCurrentTenantId();
  const canWrite = useCanWrite();
  const { isSelfHosted } = useControlPlane();
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const params = useParams({ from: appRoutes.tenantEndpointRoute.to });
  const { update } = useEndpointMutations();

  const endpointQuery = useQuery({
    ...queries.v1ServerlessEndpoints.get(params.endpoint),
    refetchInterval: ENDPOINTS_REFETCH_INTERVAL_MS,
    retry: (_failureCount, error) => shouldRetryQueryError(error),
  });

  const eventsQuery = useQuery({
    ...queries.v1ServerlessEndpoints.events(params.endpoint),
    refetchInterval: ENDPOINTS_REFETCH_INTERVAL_MS,
    enabled: !!endpointQuery.data,
  });

  const endpoint = endpointQuery.data;
  const events = useMemo(
    () => eventsQuery.data?.rows ?? [],
    [eventsQuery.data],
  );

  const diagnosis = useMemo(
    () =>
      endpoint
        ? diagnoseEndpoint({
            endpoint,
            events,
            deployment: isSelfHosted ? 'oss' : 'cloud',
          })
        : undefined,
    [endpoint, events, isSelfHosted],
  );

  if (endpointQuery.isLoading) {
    return <Loading />;
  }

  if (endpointQuery.isError) {
    if (
      isAxiosError(endpointQuery.error) &&
      endpointQuery.error.response?.status === 404
    ) {
      return (
        <ResourceNotFound
          resource={t('detail.notFound.resource')}
          primaryAction={{
            label: t('detail.notFound.back'),
            navigate: {
              to: appRoutes.tenantEndpointsRoute.to,
              params: { tenant: tenantId },
            },
          }}
        />
      );
    }

    throw endpointQuery.error;
  }

  if (!endpoint || !diagnosis) {
    return <Loading />;
  }

  const labels = Object.entries(endpoint.labels ?? {});

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto space-y-6 p-4">
        <div className="flex flex-row flex-wrap items-center justify-between gap-4">
          <div className="flex min-w-0 flex-row items-center gap-3">
            <PlatformBadge kind={endpoint.kind} className="size-6" />
            <h2 className="truncate text-2xl font-bold leading-tight text-foreground">
              {endpoint.name}
            </h2>
            <EndpointStatusBadge endpoint={endpoint} />
          </div>
          <div className="flex flex-row items-center gap-3">
            {canWrite && (
              <label className="flex flex-row items-center gap-2 text-sm">
                <Switch
                  checked={endpoint.enabled}
                  disabled={update.isPending}
                  onCheckedChange={(enabled) =>
                    update.mutate(
                      { endpointId: endpoint.metadata.id, data: { enabled } },
                      {
                        onError: (error) => handleApiError(error as AxiosError),
                      },
                    )
                  }
                />
                {t('detail.enabled')}
              </label>
            )}
            <EndpointActionsMenu
              endpoint={endpoint}
              onDeleted={() =>
                navigate({
                  to: appRoutes.tenantEndpointsRoute.to,
                  params: { tenant: tenantId },
                })
              }
            />
          </div>
        </div>

        <DiagnosisPanel
          diagnosis={diagnosis}
          rawError={endpoint.status.error}
        />

        <div className="grid gap-4 lg:grid-cols-3">
          <Card variant="light" className="border-none bg-background">
            <CardHeader>
              <CardTitle>{t('detail.configuration')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <UrlRow
                label={t('detail.healthcheckUrl')}
                url={endpoint.healthcheckUrl}
              />
              <UrlRow
                label={t('detail.triggerUrl')}
                url={endpoint.triggerUrl}
              />
              {labels.length > 0 && (
                <Row label={t('detail.labels')}>
                  <div className="flex flex-wrap gap-1">
                    {labels.map(([k, v]) => (
                      <code
                        key={k}
                        className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs"
                      >
                        {k}={String(v)}
                      </code>
                    ))}
                  </div>
                </Row>
              )}
            </CardContent>
          </Card>

          <Card variant="light" className="border-none bg-background">
            <CardHeader>
              <CardTitle>{t('detail.status')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <Row label={t('detail.health')}>
                {t(
                  endpoint.status.healthy === undefined
                    ? 'detail.health.unknown'
                    : endpoint.status.healthy
                      ? 'detail.health.healthy'
                      : 'detail.health.unhealthy',
                )}
              </Row>
              <Row label={t('detail.statusChanged')}>
                {endpoint.status.changedAt ? (
                  <RelativeDate date={endpoint.status.changedAt} />
                ) : (
                  t('detail.never')
                )}
              </Row>
              {endpoint.status.error && (
                <Row label={t('detail.lastError')}>
                  <code className="break-all font-mono text-xs text-red-600 dark:text-red-400">
                    {endpoint.status.error}
                  </code>
                </Row>
              )}
              <div className="grid grid-cols-2 gap-2">
                <Row label={t('detail.created')}>
                  <RelativeDate date={endpoint.metadata.createdAt} />
                </Row>
                <Row label={t('detail.updated')}>
                  <RelativeDate date={endpoint.metadata.updatedAt} />
                </Row>
              </div>
            </CardContent>
          </Card>

          <Card variant="light" className="border-none bg-background">
            <CardHeader>
              <CardTitle>
                {t('detail.actions.title', {
                  count: endpoint.status.registeredActions.length,
                })}
              </CardTitle>
              <CardDescription>
                <Copy k="detail.actions.description" />
              </CardDescription>
            </CardHeader>
            <CardContent className="max-h-64 space-y-3 overflow-y-auto">
              {endpoint.status.registeredActions.length === 0 ? (
                <div className="text-sm text-muted-foreground">
                  {t('detail.actions.empty')}
                </div>
              ) : (
                groupActions(endpoint.status.registeredActions).map(
                  ([workflow, actions]) => (
                    <div key={workflow}>
                      <div className="text-xs font-medium">{workflow}</div>
                      <div className="mt-1 flex flex-wrap gap-1">
                        {actions.map((a) => (
                          <code
                            key={a}
                            className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs"
                          >
                            {a}
                          </code>
                        ))}
                      </div>
                    </div>
                  ),
                )
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-3">
          <div>
            <h3 className="text-lg font-semibold">
              {t('detail.events.title')}
            </h3>
            <p className="text-sm text-muted-foreground">
              <Copy k="detail.events.description" />
            </p>
          </div>
          {eventsQuery.isError ? (
            <p className="text-sm text-muted-foreground">
              {t('detail.events.loadError')}
            </p>
          ) : (
            <EventsLog events={events} isLoading={eventsQuery.isLoading} />
          )}
        </div>
      </div>
    </div>
  );
}
