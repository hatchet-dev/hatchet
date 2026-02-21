import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import assert from '@/lib/assert';
import { useState } from 'react';

type NewTenantInputFormProps = {
  defaultTenantName?: string;
  isSaving?: boolean;
  defaultOrganizationId?: string;
  organizations?: OrganizationForUser[];
} & (
  | {
      isCloudEnabled: true;
      organizations: OrganizationForUser[];
      onSubmit: (values: {
        tenantName: string;
        organizationId: string;
      }) => void;
    }
  | {
      isCloudEnabled: false;
      onSubmit: (values: { tenantName: string }) => void;
    }
);

export function NewTenantInputForm({
  defaultTenantName = '',
  defaultOrganizationId = '',
  isSaving = false,
  isCloudEnabled,
  organizations = [],
  onSubmit,
}: NewTenantInputFormProps) {
  const [tenantName, setTenantName] = useState(defaultTenantName);
  const [organizationId, setOrganizationId] = useState(
    defaultOrganizationId || undefined,
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    assert(organizationId);
    isCloudEnabled
      ? onSubmit({ tenantName, organizationId: organizationId! })
      : onSubmit({ tenantName });
  };

  const shouldFocusOrganization = isCloudEnabled && !defaultOrganizationId;

  return (
    <form onSubmit={handleSubmit} className="grid gap-4 max-w-lg w-full">
      {isCloudEnabled && (
        <div className="grid gap-2">
          <Label htmlFor="organization-select">Organization</Label>
          <Select
            name="organizationId"
            value={organizationId}
            onValueChange={setOrganizationId}
            disabled={isSaving}
            required
          >
            <SelectTrigger
              id="organization-select"
              autoFocus={shouldFocusOrganization}
            >
              <SelectValue placeholder="Select an organization" />
            </SelectTrigger>
            <SelectContent>
              {organizations.map((org) => (
                <SelectItem key={org.metadata.id} value={org.metadata.id}>
                  {org.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className="grid gap-2">
        <Label htmlFor="tenant-name">Tenant Name</Label>
        <Input
          id="tenant-name"
          placeholder="My Tenant"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          autoFocus={!shouldFocusOrganization}
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
