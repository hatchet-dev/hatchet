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
import { useState } from 'react';
import invariant from 'tiny-invariant';

type NewTenantInputFormProps = {
  defaultTenantName?: string;
  isSaving?: boolean;
  defaultOrganizationId?: string;
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
      organizations?: null;
      onSubmit: (values: { tenantName: string }) => void;
    }
);

function OrganizationSelect({
  organizations,
  organizationId,
  setOrganizationId,
  isSaving,
  shouldFocusOrganization,
}: {
  organizations: OrganizationForUser[];
  organizationId?: string;
  setOrganizationId: (value: string) => void;
  isSaving: boolean;
  shouldFocusOrganization: boolean;
}) {
  return (
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
  );
}

export function NewTenantInputForm({
  defaultTenantName = '',
  defaultOrganizationId,
  isSaving = false,
  isCloudEnabled,
  organizations = null,
  onSubmit,
}: NewTenantInputFormProps) {
  const [tenantName, setTenantName] = useState(defaultTenantName);
  const [organizationId, setOrganizationId] = useState(
    defaultOrganizationId || undefined,
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isCloudEnabled) {
      invariant(organizationId);
      onSubmit({ tenantName, organizationId });
    } else {
      onSubmit({ tenantName });
    }
  };

  const shouldFocusOrganization = isCloudEnabled && !defaultOrganizationId;

  return (
    <form onSubmit={handleSubmit} className="grid gap-4 max-w-lg w-full">
      {!!organizations && (
        <OrganizationSelect
          organizations={organizations}
          organizationId={organizationId}
          setOrganizationId={setOrganizationId}
          isSaving={isSaving}
          shouldFocusOrganization={shouldFocusOrganization}
        />
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
