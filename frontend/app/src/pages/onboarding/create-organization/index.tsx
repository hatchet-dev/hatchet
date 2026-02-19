import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';

export default function CreateOrganization() {
  const navigate = useNavigate();

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <h1 className="text-2xl font-bold text-center">
          Create a new organization
        </h1>

        <div className="flex justify-center">
          <NewOrganizationSaverForm
            afterSave={({ organization, tenant }) => {
              navigate({
                to: appRoutes.tenantOverviewRoute.to,
                params: { tenant: tenant.id },
              });
            }}
          />
        </div>
      </div>
    </div>
  );
}
