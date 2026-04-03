import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { ArrowRightIcon } from '@radix-ui/react-icons';
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
    <form onSubmit={handleSubmit} className="grid gap-6 max-w-lg w-full">
      <div className="grid gap-2">
        <Label htmlFor="organization-name">Organization Name</Label>
        <p className="text-sm text-muted-foreground">
          Your company or team name. Used for billing and grouping your tenants
          together.
        </p>
        <Input
          id="organization-name"
          placeholder="Acme Inc."
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
        <p className="text-sm text-muted-foreground">
          An isolated environment for your tasks, workflows, workers, and
          events.
          <br />
          Most teams start with development and add a tenant for each engineer,
          staging, or production environment later.
        </p>
        <Input
          id="tenant-name"
          placeholder="development"
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
        {isSaving ? 'Getting started...' : 'Get started'}
        {!isSaving && <ArrowRightIcon className="ml-2 size-4" />}
      </Button>
    </form>
  );
}
