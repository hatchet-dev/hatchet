import { SettingsPageHeader } from '../components/settings-page-header';
import { ReadOnlyValue, SettingRow } from '../components/settings-row';
import { UpdateTenantForm } from './components/update-tenant-form';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Switch } from '@/components/v1/ui/switch';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, { UpdateTenantRequest } from '@/lib/api';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { MembershipsContextType } from '@/lib/outlet';
import { useOutletContext } from '@/lib/router-helpers';
import { type OrganizationTenantWithRegion } from '@/pages/main/v1/tenant-settings/organization';
import { TagBadge } from '@/pages/main/v1/tenant-settings/organization/components/tag-badge';
import { EditTenantTagsModal } from '@/pages/organizations/$organization/components/edit-tenant-tags-modal';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

export default function TenantSettings() {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="General"
          description="Update the tenant name and analytics preferences for this tenant."
        />

        <div className="divide-y divide-border">
          <CurrentTenant />
          <SettingRow
            label="Tenant Name"
            description="The display name for this tenant, shown across the dashboard."
          >
            <UpdateTenant />
          </SettingRow>
          <TenantApiUrl />
          <TenantRegion />
          <TenantTags />
          <SettingRow
            label="Analytics Opt-Out"
            description="Disable usage analytics collection for this tenant."
          >
            <AnalyticsOptOut />
          </SettingRow>
        </div>
      </div>
    </div>
  );
}

const CurrentTenant: React.FC = () => {
  const ctx = useOutletContext<MembershipsContextType>();
  const { organizationId } = useTenantDetails();
  const { getOrganizationIdForTenant } = useOrganizations();

  const organizationMemberships = useMemo(() => {
    const memberships = ctx?.memberships ?? [];

    if (!organizationId) {
      return memberships;
    }

    return memberships.filter(
      (m) =>
        m.tenant &&
        getOrganizationIdForTenant(m.tenant.metadata.id) === organizationId,
    );
  }, [ctx?.memberships, organizationId, getOrganizationIdForTenant]);

  return (
    <SettingRow
      label="Current Tenant"
      description={
        organizationId
          ? 'Switch between tenants in this organization.'
          : 'Switch between your tenants.'
      }
    >
      <TenantSwitcher
        memberships={organizationMemberships}
        className="w-[280px]"
      />
    </SettingRow>
  );
};

const TenantApiUrl: React.FC = () => {
  const { tenant } = useTenantDetails();
  const apiUrl = tenant?.serverUrl || window.location.origin;

  return (
    <SettingRow
      label="API URL"
      description="The base URL for this tenant's REST API."
    >
      <ReadOnlyValue value={apiUrl} />
    </SettingRow>
  );
};

const TenantRegion: React.FC = () => {
  const { tenant } = useTenantDetails();

  if (!tenant?.region) {
    return null;
  }

  return (
    <SettingRow
      label="Region"
      description="The control-plane region this tenant is deployed to."
    >
      <ReadOnlyValue value={tenant.region} />
    </SettingRow>
  );
};

// Tenant tags are a control-plane, org-owner concept (they drive which org
// members can access the tenant). Org owners can edit them here — on the
// tenant's own General tab — in addition to the org-wide Tenants list.
const TenantTags: React.FC = () => {
  const { tenant, organizationId } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const { isControlPlaneEnabled } = useControlPlane();
  const { organizations } = useOrganizations();
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);

  const isOrganizationOwner =
    organizations.find((o) => o.metadata.id === organizationId)?.isOwner ??
    false;
  const canEditTags =
    isControlPlaneEnabled && isOrganizationOwner && !!organizationId;

  const organizationQuery = useQuery({
    ...orgApi.organizationGetQuery(organizationId!),
    enabled: canEditTags,
  });
  // Widen the tag picker with tags defined on user groups but not yet applied
  // to any tenant — matches the org-side editor's suggestions.
  const userGroupsQuery = useQuery({
    ...orgApi.userGroupsListQuery(organizationId!),
    enabled: canEditTags,
  });

  if (!canEditTags) {
    return null;
  }

  // Cloud's OrganizationTenant type omits `tags`; the control-plane data has
  // it. Same graft the org-wide Tenants list uses.
  const tenants: OrganizationTenantWithRegion[] =
    organizationQuery.data?.tenants ?? [];
  const tags = tenants.find((t) => t.id === tenantId)?.tags ?? [];

  const tagSet = new Set<string>();
  for (const t of tenants) {
    t.tags?.forEach((tag) => tagSet.add(tag));
  }
  for (const group of userGroupsQuery.data ?? []) {
    group.tags?.forEach((tag) => tagSet.add(tag));
  }
  const allTenantTags = Array.from(tagSet).sort();

  return (
    <SettingRow
      label="Tags"
      description="Tags control which organization members can access this tenant."
    >
      <div className="flex items-center gap-3">
        <div className="flex flex-wrap items-center gap-1">
          {tags.length > 0 ? (
            tags.map((tag) => <TagBadge key={tag} tag={tag} />)
          ) : (
            <span className="text-sm text-muted-foreground">No tags</span>
          )}
        </div>
        <Button
          variant="outline"
          size="sm"
          className="shrink-0"
          onClick={() => setIsEditing(true)}
        >
          Edit tags
        </Button>
      </div>

      {isEditing && (
        <EditTenantTagsModal
          open={isEditing}
          onOpenChange={(open) => !open && setIsEditing(false)}
          organizationId={organizationId!}
          tenantId={tenantId}
          tenantName={tenant?.name || tenantId}
          initialTags={tags}
          allTenantTags={allTenantTags}
          onSuccess={() =>
            queryClient.invalidateQueries({
              queryKey: ['organization:get', organizationId],
            })
          }
        />
      )}
    </SettingRow>
  );
};

const UpdateTenant: React.FC = () => {
  const [isLoading, setIsLoading] = useState(false);
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onMutate: () => setIsLoading(true),
    onSuccess: () => window.location.reload(),
    onError: handleApiError,
  });

  return (
    <UpdateTenantForm
      isLoading={isLoading}
      onSubmit={(data) => updateMutation.mutate(data)}
    />
  );
};

const AnalyticsOptOut: React.FC = () => {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const [changed, setChanged] = useState(false);
  const [checkedState, setChecked] = useState(!!tenant?.analyticsOptOut);
  const [isLoading, setIsLoading] = useState(false);
  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onMutate: () => setIsLoading(true),
    onSuccess: () => window.location.reload(),
    onSettled: () => setTimeout(() => setIsLoading(false), 1000),
    onError: handleApiError,
  });

  return (
    <div className="flex items-center gap-3">
      <Switch
        id="aoo"
        checked={checkedState}
        onClick={() => {
          setChecked((s) => !s);
          setChanged(true);
        }}
      />
      {changed &&
        (isLoading ? (
          <Spinner />
        ) : (
          <Button
            size="sm"
            onClick={() =>
              updateMutation.mutate({ analyticsOptOut: checkedState })
            }
          >
            Save
          </Button>
        ))}
    </div>
  );
};
