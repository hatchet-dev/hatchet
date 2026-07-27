import { SettingsPageHeader } from '../components/settings-page-header';
import { TenantResourceLimitsTable } from './components/tenant-resource-limits-table';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';

export default function ResourceLimits() {
  const { tenant } = useParams({
    from: appRoutes.tenantSettingsResourceLimitsRoute.to,
  });

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Resource Limits"
          description="Review the resource limits currently applied to this tenant."
        />

        <TenantResourceLimitsTable tenantId={tenant} showDocsOnEmpty />
      </div>
    </div>
  );
}
