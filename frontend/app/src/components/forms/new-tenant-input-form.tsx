import {
  RegionSelect,
  shardDeploymentKey,
} from '@/components/forms/region-select';
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
import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';
import { useMemo, useState } from 'react';
import invariant from 'tiny-invariant';

type NewTenantInputFormProps = {
  defaultTenantName?: string;
  isSaving?: boolean;
} & (
  | {
      isCloudEnabled: true;
      organizations: OrganizationForUser[];
      organizationId?: string;
      onOrganizationIdChange: (organizationId: string) => void;
      onSubmit: (values: {
        tenantName: string;
        organizationId: string;
        region?: string;
      }) => void;
      showRegionSelect?: boolean;
      availableShards?: OrganizationAvailableShard[];
      isShardsLoading?: boolean;
    }
  | {
      isCloudEnabled: false;
      organizations?: null;
      onSubmit: (values: { tenantName: string }) => void;
      organizationId?: undefined;
      onOrganizationIdChange?: undefined;
      showRegionSelect?: false;
      availableShards?: undefined;
      isShardsLoading?: false;
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
  isSaving = false,
  isCloudEnabled,
  organizations = null,
  organizationId,
  onOrganizationIdChange,
  onSubmit,
  showRegionSelect = false,
  availableShards = [],
  isShardsLoading = false,
}: NewTenantInputFormProps) {
  const [tenantName, setTenantName] = useState(defaultTenantName);
  const [selectedDeploymentRegion, setSelectedDeploymentRegion] = useState<
    string | undefined
  >();

  const shardKeys = useMemo(
    () => availableShards.map(shardDeploymentKey),
    [availableShards],
  );

  const deploymentRegion =
    selectedDeploymentRegion && shardKeys.includes(selectedDeploymentRegion)
      ? selectedDeploymentRegion
      : shardKeys[0];

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isCloudEnabled) {
      invariant(organizationId);
      onSubmit({
        tenantName,
        organizationId,
        ...(showRegionSelect && deploymentRegion
          ? { region: deploymentRegion }
          : {}),
      });
    } else {
      onSubmit({ tenantName });
    }
  };

  const shouldFocusOrganization = isCloudEnabled && !organizationId;

  const cannotSubmitRegion =
    showRegionSelect &&
    (isShardsLoading ||
      availableShards.length === 0 ||
      (availableShards.length > 0 && !deploymentRegion));

  return (
    <form onSubmit={handleSubmit} className="grid gap-6 max-w-lg w-full">
      {isCloudEnabled && organizations && (
        <OrganizationSelect
          organizations={organizations}
          organizationId={organizationId}
          setOrganizationId={onOrganizationIdChange}
          isSaving={isSaving}
          shouldFocusOrganization={shouldFocusOrganization}
        />
      )}

      {isCloudEnabled &&
        showRegionSelect &&
        (isShardsLoading || availableShards.length > 0) && (
          <RegionSelect
            shards={availableShards}
            value={deploymentRegion}
            onValueChange={setSelectedDeploymentRegion}
            isLoading={isShardsLoading}
          />
        )}

      <div className="grid gap-2">
        <Label htmlFor="tenant-name">Tenant Name</Label>
        <p className="text-sm text-muted-foreground">
          An isolated environment for your workflows, workers, and events. Most
          teams start with dev and add staging or production later.
        </p>
        <Input
          id="tenant-name"
          placeholder="production"
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

      <Button
        type="submit"
        className="w-full"
        disabled={isSaving || cannotSubmitRegion}
      >
        {isSaving ? 'Getting started...' : 'Get started'}
      </Button>
    </form>
  );
}
