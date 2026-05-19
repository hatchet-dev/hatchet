import { CloudOrganizationSettings } from '@/pages/main/v1/tenant-settings/organization';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';

export default function OrganizationPage() {
  const { organization } = useParams({
    from: appRoutes.organizationsRoute.to,
  });

  return <CloudOrganizationSettings orgId={organization} />;
}
