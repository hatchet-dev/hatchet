import { DocsButton } from '@/components/v1/docs/docs-button';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { docsPages } from '@/lib/generated/docs';
import { appRoutes } from '@/router';
import { ShieldCheckIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

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

export function AuditLogSettings({ orgId }: { orgId: string }) {
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const orgApi = useOrganizationApi();

  const entitlementsQuery = useQuery({
    ...orgApi.organizationEntitlementsGetQuery(orgId),
    enabled: !!orgId && isControlPlaneEnabled,
  });

  if (isControlPlaneLoading || entitlementsQuery.isLoading) {
    return <Spinner />;
  }

  if (entitlementsQuery.data?.auditLogs !== true) {
    return <AuditLogUpgrade orgId={orgId} />;
  }

  return <AuditLogEnabled orgId={orgId} />;
}

function AuditLogRetrieval({ orgId }: { orgId: string }) {
  const endpoint = `${window.location.origin}/api/v1/control-plane/organizations/${orgId}/audit-logs`;
  const curlExample = `curl \\
  -H "Authorization: Bearer <MANAGEMENT_TOKEN>" \\
  "${endpoint}?limit=100"`;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Endpoint</p>
        <CodeHighlighter language="text" code={endpoint} wrapLines={false} />
        <p className="text-sm text-muted-foreground">
          Requests must be authenticated with a Management Token (sent as a
          Bearer token). Create one in the{' '}
          <span className="font-medium text-foreground">Management Tokens</span>{' '}
          tab, then call this endpoint to retrieve your organization's audit
          logs.
        </p>
      </div>
      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Example request</p>
        <CodeHighlighter language="text" code={curlExample} />
      </div>
      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Query parameters</p>
        <ul className="space-y-1 text-sm text-muted-foreground">
          <li>
            <code className="text-foreground">tenant</code> &mdash; optional
            tenant ID to scope results to a single tenant
          </li>
          <li>
            <code className="text-foreground">limit</code> &mdash; max rows to
            return (default 1000, max 1000)
          </li>
          <li>
            <code className="text-foreground">offset</code> &mdash; number of
            rows to skip for pagination (default 0)
          </li>
          <li>
            <code className="text-foreground">since</code> &mdash; RFC3339
            timestamp for the start of the range (default 24 hours ago)
          </li>
          <li>
            <code className="text-foreground">until</code> &mdash; RFC3339
            timestamp for the end of the range (default now)
          </li>
        </ul>
      </div>
    </div>
  );
}

function AuditLogEnabled({ orgId }: { orgId: string }) {
  return (
    <div>
      <SectionHeader
        title="Audit Log"
        description="Retrieve an immutable record of actions taken across your organization's tenants for compliance and security review."
      />
      <Separator className="my-4" />
      <AuditLogRetrieval orgId={orgId} />
      <Separator className="my-6" />
      <p className="text-sm text-muted-foreground">
        See the{' '}
        <DocsButton
          doc={docsPages.v1.security['audit-logs']}
          label="audit logs documentation"
          variant="text"
        />{' '}
        for the full response schema and details.
      </p>
    </div>
  );
}

function AuditLogUpgrade({ orgId }: { orgId: string }) {
  const navigate = useNavigate();

  return (
    <div className="py-12">
      <EmptyState
        graphic={
          <div className="rounded-full bg-primary/10 p-3">
            <ShieldCheckIcon className="h-8 w-8 text-primary" />
          </div>
        }
        title="Unlock Audit Logs"
        description="Audit logs give you an immutable record of actions taken across your organization's tenants for compliance and security review. Upgrade your plan to enable this feature."
        buttons={[
          {
            label: 'View plans',
            onClick: () =>
              navigate({
                to: appRoutes.organizationBillingRoute.to,
                params: { organization: orgId },
              }),
          },
        ]}
      />
    </div>
  );
}
