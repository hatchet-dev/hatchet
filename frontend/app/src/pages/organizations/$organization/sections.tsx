import {
  OrganizationSettings,
  OrganizationSettingsSection,
} from '@/pages/main/v1/tenant-settings/organization';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';

function OrganizationSectionPage({
  section,
}: {
  section: OrganizationSettingsSection;
}) {
  const { organization } = useParams({
    from: appRoutes.organizationsRoute.to,
  });

  return <OrganizationSettings orgId={organization} section={section} />;
}

export function OrganizationTenantsPage() {
  return <OrganizationSectionPage section="tenants" />;
}

export function OrganizationTeamPage() {
  return <OrganizationSectionPage section="team" />;
}

export function OrganizationTokensPage() {
  return <OrganizationSectionPage section="tokens" />;
}

export function OrganizationRegionsPage() {
  return <OrganizationSectionPage section="regions" />;
}

export function OrganizationSsoPage() {
  return <OrganizationSectionPage section="sso" />;
}

export function OrganizationAuditLogPage() {
  return <OrganizationSectionPage section="audit-log" />;
}
