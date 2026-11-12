import { SettingRow } from '../components/settings-row';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
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

  return <AuditLogEnabled />;
}

function AuditLogEnabled() {
  return (
    <div>
      <SettingRow
        label="Audit Logs"
        description="An immutable record of actions taken across your organization's tenants for compliance and security review."
      >
        <DocsButton
          doc={docsPages.v1.security['audit-logs']}
          label="View docs"
        />
      </SettingRow>
      <p className="pb-4 text-sm text-muted-foreground">
        To get an export of your organization's audit logs, contact{' '}
        <a href="mailto:support@hatchet.run" className="text-primary underline">
          support@hatchet.run
        </a>
        .
      </p>
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
