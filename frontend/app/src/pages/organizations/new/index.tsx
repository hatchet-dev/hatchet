import { OrganizationAndTenantForm } from '@/components/forms/organization-and-tenant-form';
import { Button } from '@/components/v1/ui/button';
import { XMarkIcon } from '@heroicons/react/24/outline';

export default function OrganizationsNew() {
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
          <OrganizationAndTenantForm
            isCloudEnabled={true}
            onSubmit={(values) => {
              console.log(values);
            }}
          />
        </div>
      </div>
    </div>
  );
}
