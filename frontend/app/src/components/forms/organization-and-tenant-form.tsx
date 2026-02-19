import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { useMemo, useState } from 'react';

type OrganizationAndTenantFormProps = {
  defaultOrganizationName?: string;
  defaultTenantName?: string;
} & (
  | {
      isCloudEnabled: true;
      onSubmit: (values: {
        organizationName: string;
        tenantName: string;
      }) => void;
    }
  | {
      isCloudEnabled: false;
      onSubmit: (values: { tenantName: string }) => void;
    }
);

export function OrganizationAndTenantForm({
  defaultOrganizationName = '',
  defaultTenantName = '',
  onSubmit,
  isCloudEnabled,
}: OrganizationAndTenantFormProps) {
  const [organizationName, setOrganizationName] = useState(
    defaultOrganizationName,
  );
  const [tenantName, setTenantName] = useState(defaultTenantName);

  const submit = isCloudEnabled
    ? () => onSubmit({ organizationName, tenantName })
    : () => onSubmit({ tenantName });

  const canSubmit = useMemo(
    () => !!(isCloudEnabled ? organizationName && tenantName : tenantName),
    [isCloudEnabled, organizationName, tenantName],
  );

  return (
    <div className="grid gap-4 max-w-lg w-full">
      <div className="grid gap-2">
        <Label htmlFor="organization-name">Organization Name</Label>
        <Input
          id="organization-name"
          placeholder="My Organization"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          autoFocus={true}
          spellCheck={false}
          value={organizationName}
          onChange={(e) => setOrganizationName(e.target.value)}
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="tenant-name">Tenant Name</Label>
        <Input
          id="tenant-name"
          placeholder="My Tenant"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          value={tenantName}
          onChange={(e) => setTenantName(e.target.value)}
        />
      </div>

      <Button className="w-full" onClick={submit} disabled={!canSubmit}>
        Create
      </Button>
    </div>
  );
}
