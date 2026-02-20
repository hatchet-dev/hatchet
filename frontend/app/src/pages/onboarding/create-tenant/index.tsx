import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';

export default function CreateTenant() {
  const navigate = useNavigate();

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <h1 className="text-2xl font-bold text-center">
          Create a new tenant
        </h1>

        <div className="flex justify-center">
          <NewTenantSaverForm
            afterSave={(result) => {
              const tenantId =
                result.type === 'cloud'
                  ? result.tenant.id
                  : result.tenant.metadata.id;
              navigate({
                to: appRoutes.tenantOverviewRoute.to,
                params: { tenant: tenantId },
              });
            }}
          />
        </div>
      </div>
    </div>
  );
}
