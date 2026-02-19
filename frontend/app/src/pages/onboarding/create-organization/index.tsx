import { OrganizationAndTenantForm } from '@/components/forms/organization-and-tenant-form';

export default function CreateOrganization() {
  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
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
