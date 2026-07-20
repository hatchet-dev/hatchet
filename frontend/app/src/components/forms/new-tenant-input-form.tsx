import { RegionSelect } from '@/components/forms/region-select';
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
import { OrganizationForUser } from '@/lib/api/generated/control-plane/data-contracts';
import { OrganizationAvailableShard } from '@/lib/api/generated/control-plane/data-contracts';
import { shardDeploymentKey } from '@/lib/shard-deployment-key';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { KeyboardEvent, useCallback, useMemo, useState } from 'react';
import invariant from 'tiny-invariant';

type NewTenantInputFormProps = {
  defaultTenantName?: string;
  isSaving?: boolean;
  allTenantTags?: string[];
} & (
  | {
      isControlPlaneEnabled: true;
      organizations: OrganizationForUser[];
      organizationId?: string;
      onOrganizationIdChange: (organizationId: string) => void;
      onSubmit: (values: {
        tenantName: string;
        organizationId: string;
        region?: string;
        tags?: string[];
      }) => void;
      showRegionSelect?: boolean;
      availableShards?: OrganizationAvailableShard[];
      isShardsLoading?: boolean;
      showTagsInput?: boolean;
    }
  | {
      isControlPlaneEnabled: false;
      organizations?: null;
      onSubmit: (values: { tenantName: string }) => void;
      organizationId?: undefined;
      onOrganizationIdChange?: undefined;
      showRegionSelect?: false;
      availableShards?: undefined;
      isShardsLoading?: false;
      showTagsInput?: false;
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
  isControlPlaneEnabled,
  organizations = null,
  organizationId,
  onOrganizationIdChange,
  onSubmit,
  showRegionSelect = false,
  availableShards = [],
  isShardsLoading = false,
  showTagsInput = false,
  allTenantTags = [],
}: NewTenantInputFormProps) {
  const [tenantName, setTenantName] = useState(defaultTenantName);
  const [selectedDeploymentRegion, setSelectedDeploymentRegion] = useState<
    string | undefined
  >();
  const [tags, setTags] = useState<string[]>([]);
  const [tagInputValue, setTagInputValue] = useState('');

  const shardKeys = useMemo(
    () => availableShards.map(shardDeploymentKey),
    [availableShards],
  );

  const deploymentRegion =
    selectedDeploymentRegion && shardKeys.includes(selectedDeploymentRegion)
      ? selectedDeploymentRegion
      : shardKeys[0];

  const addTag = useCallback(
    (value: string) => {
      const trimmed = value.trim();
      if (trimmed && !tags.includes(trimmed)) {
        setTags((prev) => [...prev, trimmed]);
      }
      setTagInputValue('');
    },
    [tags],
  );

  const removeTag = (tag: string) =>
    setTags((prev) => prev.filter((t) => t !== tag));

  const handleTagKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag(tagInputValue);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isControlPlaneEnabled) {
      invariant(organizationId);
      onSubmit({
        tenantName,
        organizationId,
        ...(showRegionSelect && deploymentRegion
          ? { region: deploymentRegion }
          : {}),
        ...(showTagsInput && tags.length > 0 ? { tags } : {}),
      });
    } else {
      onSubmit({ tenantName });
    }
  };

  const shouldFocusOrganization = isControlPlaneEnabled && !organizationId;

  const cannotSubmitRegion =
    showRegionSelect &&
    (isShardsLoading ||
      availableShards.length === 0 ||
      (availableShards.length > 0 && !deploymentRegion));

  return (
    <form onSubmit={handleSubmit} className="grid gap-6 max-w-lg w-full">
      {isControlPlaneEnabled && organizations && (
        <OrganizationSelect
          organizations={organizations}
          organizationId={organizationId}
          setOrganizationId={onOrganizationIdChange}
          isSaving={isSaving}
          shouldFocusOrganization={shouldFocusOrganization}
        />
      )}

      {isControlPlaneEnabled &&
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

      {showTagsInput && (
        <div className="grid gap-2">
          <Label>Tags (optional)</Label>
          <p className="text-sm text-muted-foreground">
            Users in user groups which match these tags automatically get access
            to this tenant.
          </p>
          {tags.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1.5 rounded-md border bg-secondary px-3 py-1 text-sm text-secondary-foreground"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() => removeTag(tag)}
                    disabled={isSaving}
                    className="ml-0.5 rounded hover:text-destructive disabled:opacity-50"
                    aria-label={`Remove ${tag}`}
                  >
                    <XMarkIcon className="size-3.5" />
                  </button>
                </span>
              ))}
            </div>
          )}
          {allTenantTags.filter((t) => !tags.includes(t)).length > 0 && (
            <Select onValueChange={addTag} disabled={isSaving} value="">
              <SelectTrigger>
                <SelectValue placeholder="Add an existing tag…" />
              </SelectTrigger>
              <SelectContent>
                {allTenantTags
                  .filter((t) => !tags.includes(t))
                  .map((tag) => (
                    <SelectItem key={tag} value={tag}>
                      {tag}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          )}
          <div className="flex gap-2">
            <Input
              id="tenant-tags"
              placeholder="New tag..."
              type="text"
              value={tagInputValue}
              onChange={(e) => setTagInputValue(e.target.value)}
              onKeyDown={handleTagKeyDown}
              disabled={isSaving}
              autoComplete="off"
            />
            <Button
              type="button"
              variant="outline"
              onClick={() => addTag(tagInputValue)}
              disabled={isSaving || !tagInputValue.trim()}
            >
              Add
            </Button>
          </div>
        </div>
      )}

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
