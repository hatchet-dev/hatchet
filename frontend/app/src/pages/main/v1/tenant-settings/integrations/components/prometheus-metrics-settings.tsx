import { UpgradeRequired } from '@/components/v1/cloud/upgrade-required';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantDetails } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
import { ChartBarIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';

const CONFIG_DOCS_URL =
  'https://docs.hatchet.run/self-hosting/configuration-options';

function SectionHeader({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="mb-4 space-y-1">
      <h2 className="text-sm font-medium text-foreground">{title}</h2>
      <p className="max-w-2xl text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

export function PrometheusMetricsSettings() {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const { organizationId } = useTenantDetails();
  const orgApi = useOrganizationApi();

  const entitlementsQuery = useQuery({
    ...orgApi.organizationEntitlementsGetQuery(organizationId || ''),
    enabled: !!organizationId && isControlPlaneEnabled,
  });

  if (isControlPlaneLoading) {
    return <Spinner />;
  }

  if (isControlPlaneEnabled) {
    if (!organizationId || entitlementsQuery.isLoading) {
      return <Spinner />;
    }

    if (entitlementsQuery.data?.prometheusMetrics !== true) {
      return <MetricsUpgrade organizationId={organizationId} />;
    }

    return <MetricsEnabled tenantId={tenantId} />;
  }

  return <MetricsSelfHostInstructions tenantId={tenantId} />;
}

function resolveTenantApiTarget(serverUrl?: string) {
  // The tenant's API may live on a different host than the dashboard (e.g. a
  // control-plane shard), so prefer the tenant's serverUrl and fall back to the
  // current window location for self-hosted deployments where it is unset.
  if (serverUrl) {
    try {
      const url = new URL(serverUrl);
      return {
        origin: url.origin,
        host: url.host,
        scheme: url.protocol.replace(':', ''),
      };
    } catch {
      // Malformed serverUrl; fall through to window.location.
    }
  }

  return {
    origin: window.location.origin,
    host: window.location.host,
    scheme: window.location.protocol.replace(':', ''),
  };
}

function MetricsScrapeConfig({ tenantId }: { tenantId: string }) {
  const { tenant } = useTenantDetails();
  const { origin, host, scheme } = resolveTenantApiTarget(tenant?.serverUrl);

  const scrapeUrl = `${origin}/api/v1/tenants/${tenantId}/prometheus-metrics`;
  const scrapeConfig = `scrape_configs:
  - job_name: hatchet
    scheme: ${scheme}
    metrics_path: /api/v1/tenants/${tenantId}/prometheus-metrics
    authorization:
      type: Bearer
      credentials: <HATCHET_API_TOKEN>
    static_configs:
      - targets:
          - ${host}`;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Scrape URL</p>
        <CodeHighlighter language="text" code={scrapeUrl} wrapLines={false} />
        <p className="text-sm text-muted-foreground">
          Requests must be authenticated with a{' '}
          <Link
            to={appRoutes.tenantSettingsApiTokensRoute.to}
            params={{ tenant: tenantId }}
            className="underline"
          >
            tenant API token
          </Link>{' '}
          (sent as a Bearer token). Point your Prometheus instance or Grafana
          Agent at this URL to federate this tenant's metrics.
        </p>
      </div>
      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">
          Example Prometheus scrape config
        </p>
        <CodeHighlighter language="yaml" code={scrapeConfig} />
      </div>
    </div>
  );
}

function MetricsPreview({ tenantId }: { tenantId: string }) {
  const preview = useQuery({
    queryKey: ['tenant:prometheus-metrics', tenantId],
    queryFn: async () => (await api.tenantGetPrometheusMetrics(tenantId)).data,
    retry: false,
  });

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-foreground">Live preview</p>
      {preview.isLoading ? (
        <Spinner />
      ) : preview.isError || !preview.data ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No metrics available yet. Once your workers report metrics, they will
          appear here.
        </div>
      ) : (
        <CodeHighlighter
          language="text"
          code={preview.data}
          maxHeight="24rem"
          copy={false}
          wrapLines={false}
        />
      )}
    </div>
  );
}

function MetricsEnabled({ tenantId }: { tenantId: string }) {
  return (
    <div>
      <SectionHeader
        title="Prometheus Metrics"
        description="Scrape this tenant's metrics in Prometheus exposition format to power external dashboards and alerting."
      />
      <Separator className="my-4" />
      <MetricsScrapeConfig tenantId={tenantId} />
      <Separator className="my-6" />
      <MetricsPreview tenantId={tenantId} />
    </div>
  );
}

function MetricsSelfHostInstructions({ tenantId }: { tenantId: string }) {
  const { meta } = useApiMeta();
  const prometheusServerEnabled =
    !!meta &&
    'prometheusServerEnabled' in meta &&
    !!meta.prometheusServerEnabled;

  return (
    <div>
      <SectionHeader
        title="Prometheus Metrics"
        description="Expose this tenant's metrics in Prometheus exposition format for external dashboards and alerting."
      />
      <Separator className="my-4" />
      <div className="space-y-4">
        <p className="text-sm text-muted-foreground">
          To enable tenant-scoped Prometheus metrics on a self-hosted Hatchet
          server, configure the following environment variables on the API
          server:
        </p>
        <CodeHighlighter
          language="text"
          code={`SERVER_PROMETHEUS_SERVER_URL=<your-prometheus-federation-url>
SERVER_PROMETHEUS_SERVER_USERNAME=<optional-basic-auth-username>
SERVER_PROMETHEUS_SERVER_PASSWORD=<optional-basic-auth-password>
# Gate the endpoint per tenant on the prometheus_metrics entitlement
SERVER_PROMETHEUS_SERVER_TENANT_SCOPED=true`}
        />
        <p className="text-sm text-muted-foreground">
          See the{' '}
          <a
            href={CONFIG_DOCS_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="underline"
          >
            configuration options documentation
          </a>{' '}
          for details.
        </p>
      </div>
      {prometheusServerEnabled && (
        <>
          <Separator className="my-6" />
          <MetricsScrapeConfig tenantId={tenantId} />
          <Separator className="my-6" />
          <MetricsPreview tenantId={tenantId} />
        </>
      )}
    </div>
  );
}

function MetricsUpgrade({ organizationId }: { organizationId: string }) {
  return (
    <UpgradeRequired
      icon={<ChartBarIcon className="h-8 w-8 text-primary" />}
      title="Unlock Prometheus metrics"
      description="Prometheus metrics let you federate this tenant's metrics into your own dashboards and alerting. Upgrade your plan to enable this feature."
      action={
        <Link
          to={appRoutes.organizationBillingRoute.to}
          params={{ organization: organizationId }}
          className="w-full"
        >
          <Button className="min-w-40 px-8 py-6 text-base" size="lg">
            View plans
          </Button>
        </Link>
      }
    />
  );
}
