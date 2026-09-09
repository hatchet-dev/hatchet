import { SettingRow } from '../components/settings-row';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { OFFICE_HOURS_URL } from '@/lib/external-links';
import { docsPages } from '@/lib/generated/docs';
import { useQuery } from '@tanstack/react-query';

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
    return <AuditLogUpgrade />;
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
      <SettingRow
        label="Audit Logs"
        description="Retrieve an immutable record of actions taken across your organization's tenants for compliance and security review."
      >
        <DocsButton
          doc={docsPages.v1.security['audit-logs']}
          label="View docs"
        />
      </SettingRow>
      <div className="pb-4">
        <AuditLogRetrieval orgId={orgId} />
      </div>
    </div>
  );
}

function AuditLogUpgrade() {
  return (
    <SettingRow
      label="Audit Logs"
      description="An immutable record of every administrative action across your organization's tenants: who did what and when. Evidence for SOC 2 audits, security reviews, and incident investigation. Included on Hatchet Custom plans."
    >
      <Button
        className="shrink-0"
        onClick={() => window.open(OFFICE_HOURS_URL, '_blank', 'noreferrer')}
      >
        Talk to us
      </Button>
      <DocsButton doc={docsPages.v1.security['audit-logs']} label="View docs" />
    </SettingRow>
  );
}
