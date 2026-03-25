import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useNavigate } from '@tanstack/react-router';

export default function OrganizationsNew() {
  const navigate = useNavigate();

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <div className="flex justify-end">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.history.back()}
            className="h-8 w-8 p-0"
          >
            <XMarkIcon className="size-4" />
          </Button>
        </div>

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
