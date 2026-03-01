import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { useCallback, useState } from 'react';

type NewOrganizationInputFormProps = {
  defaultOrganizationName?: string;
  defaultTenantName?: string;
  isSaving: boolean;
  onSubmit: (values: { organizationName: string; tenantName: string }) => void;
};

export function NewOrganizationInputForm({
  defaultOrganizationName = '',
  defaultTenantName = '',
  onSubmit,
  isSaving,
}: NewOrganizationInputFormProps) {
  const [organizationName, setOrganizationName] = useState(
    defaultOrganizationName,
  );
  const [tenantName, setTenantName] = useState(defaultTenantName);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onSubmit({ organizationName, tenantName });
    },
    [organizationName, tenantName, onSubmit],
  );

  return (
    <form onSubmit={handleSubmit} className="grid gap-4 max-w-lg w-full">
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
          disabled={isSaving}
          required
        />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="tenant-name">Name of First Tenant</Label>
        <Input
          id="tenant-name"
          placeholder="My Tenant"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          value={tenantName}
          onChange={(e) => setTenantName(e.target.value)}
          disabled={isSaving}
          required
        />
      </div>

      <Button type="submit" className="w-full" disabled={isSaving}>
        {isSaving ? 'Creating...' : 'Create'}
      </Button>
    </form>
  );
}
