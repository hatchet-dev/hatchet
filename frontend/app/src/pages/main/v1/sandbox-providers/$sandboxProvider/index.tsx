import { EventsLog } from '../components/events-log';
import { SandboxGate } from '../components/gate';
import { ProviderActionsMenu } from '../components/provider-actions';
import { ProviderStatusBadge } from '../components/provider-status-badge';
import {
  SANDBOX_PROVIDERS_REFETCH_INTERVAL_MS,
  useSandboxProviderMutations,
} from '../hooks/use-sandbox-providers';
import { KINDS } from '../lib/kinds';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Loading } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import useCanWrite from '@/hooks/use-can-write';
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

export default function SandboxProviderDetail() {
  return (
    <SandboxGate>
      <SandboxProviderDetailContent />
    </SandboxGate>
  );
}

function SandboxProviderDetailContent() {
  const { tenantId } = useCurrentTenantId();
  const canWrite = useCanWrite();
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const params = useParams({ from: appRoutes.tenantSandboxProviderRoute.to });
  const { update } = useSandboxProviderMutations();

  const providerQuery = useQuery({
    ...queries.cloud.getSandboxProvider(params.sandboxProvider),
    refetchInterval: SANDBOX_PROVIDERS_REFETCH_INTERVAL_MS,
    retry: (_failureCount, error) => shouldRetryQueryError(error),
  });

  const eventsQuery = useQuery({
    ...queries.cloud.listSandboxProviderEvents(params.sandboxProvider),
    refetchInterval: SANDBOX_PROVIDERS_REFETCH_INTERVAL_MS,
    enabled: !!providerQuery.data,
  });

  const provider = providerQuery.data;
  const events = useMemo(
    () => eventsQuery.data?.rows ?? [],
    [eventsQuery.data],
  );

  if (providerQuery.isLoading) {
    return <Loading />;
  }

  if (providerQuery.isError) {
    if (
      isAxiosError(providerQuery.error) &&
      providerQuery.error.response?.status === 404
    ) {
      return (
        <ResourceNotFound
          resource="Sandbox provider"
          primaryAction={{
            label: 'Back to sandbox providers',
            navigate: {
              to: appRoutes.tenantSandboxProvidersRoute.to,
              params: { tenant: tenantId },
            },
          }}
        />
      );
    }

    throw providerQuery.error;
  }

  if (!provider) {
    return <Loading />;
  }

  const labels = Object.entries(provider.labels ?? {});
  const config = Object.entries(provider.config ?? {});

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto space-y-6 p-4">
        <div className="flex flex-row flex-wrap items-center justify-between gap-4">
          <div className="flex min-w-0 flex-row items-center gap-3">
            <Badge variant="outline">{KINDS[provider.kind].label}</Badge>
            <h2 className="truncate text-2xl font-bold leading-tight text-foreground">
              {provider.name}
            </h2>
            <ProviderStatusBadge provider={provider} />
          </div>
          <div className="flex flex-row items-center gap-3">
            {canWrite && (
              <label className="flex flex-row items-center gap-2 text-sm">
                <Switch
                  checked={provider.enabled}
                  disabled={update.isPending}
                  onCheckedChange={(enabled) =>
                    update.mutate(
                      { providerId: provider.metadata.id, data: { enabled } },
                      {
                        onError: (error) => handleApiError(error as AxiosError),
                      },
                    )
                  }
                />
                Enabled
              </label>
            )}
            <ProviderActionsMenu
              provider={provider}
              onDeleted={() =>
                navigate({
                  to: appRoutes.tenantSandboxProvidersRoute.to,
                  params: { tenant: tenantId },
                })
              }
            />
          </div>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card variant="light" className="border-none bg-background">
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <Row label="Default template">
                {provider.defaultTemplate ? (
                  <code className="font-mono text-xs">
                    {provider.defaultTemplate}
                  </code>
                ) : (
                  <span className="text-muted-foreground">Vendor default</span>
                )}
              </Row>
              <Row label="Max idle time">
                {provider.maxIdleTtlSeconds} seconds
              </Row>
              {config.length > 0 && (
                <Row label="Provider config">
                  <pre className="overflow-x-auto rounded bg-muted px-2 py-1 font-mono text-xs">
                    {JSON.stringify(provider.config, null, 2)}
                  </pre>
                </Row>
              )}
              {labels.length > 0 && (
                <Row label="Labels">
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
              <CardTitle>Status</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <Row label="Credentials">
                {provider.status.healthy === undefined
                  ? 'Not checked yet'
                  : provider.status.healthy
                    ? 'Accepted by the vendor'
                    : 'Rejected by the vendor'}
              </Row>
              <Row label="Status changed">
                {provider.status.changedAt ? (
                  <RelativeDate date={provider.status.changedAt} />
                ) : (
                  'Never'
                )}
              </Row>
              {provider.status.error && (
                <Row label="Last error">
                  <code className="break-all font-mono text-xs text-red-600 dark:text-red-400">
                    {provider.status.error}
                  </code>
                </Row>
              )}
              <div className="grid grid-cols-2 gap-2">
                <Row label="Created">
                  <RelativeDate date={provider.metadata.createdAt} />
                </Row>
                <Row label="Updated">
                  <RelativeDate date={provider.metadata.updatedAt} />
                </Row>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-3">
          <div>
            <h3 className="text-lg font-semibold">Events</h3>
            <p className="text-sm text-muted-foreground">
              What the sandbox operator observed about this provider: credential
              checks, sandbox creations and failures.
            </p>
          </div>
          {eventsQuery.isError ? (
            <p className="text-sm text-muted-foreground">
              Events could not be loaded.
            </p>
          ) : (
            <EventsLog events={events} isLoading={eventsQuery.isLoading} />
          )}
        </div>
      </div>
    </div>
  );
}
