import { SettingsPageHeader } from '../components/settings-page-header';
import { ReadOnlyValue, SettingRow } from '../components/settings-row';
import { UpdateTenantForm } from './components/update-tenant-form';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
import useCanWrite from '@/hooks/use-can-write';
import useControlPlane from '@/hooks/use-control-plane';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, { UpdateTenantRequest } from '@/lib/api';
import { TenantStatusType } from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { MembershipsContextType } from '@/lib/outlet';
import { useOutletContext } from '@/lib/router-helpers';
import {
  defaultOnboardingState,
  onboardingStorageKey,
} from '@/pages/main/v1/overview/components/onboarding-state';
import { type OrganizationTenantWithRegion } from '@/pages/main/v1/tenant-settings/organization';
import { TagBadge } from '@/pages/main/v1/tenant-settings/organization/components/tag-badge';
import { DeleteTenantModal } from '@/pages/organizations/$organization/components/delete-tenant-modal';
import { EditTenantTagsModal } from '@/pages/organizations/$organization/components/edit-tenant-tags-modal';
import { TransferTenantModal } from '@/pages/organizations/$organization/components/transfer-tenant-modal';
import { appRoutes } from '@/router';
import { ArrowsRightLeftIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
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
          <OnboardingSettingRow />
        </div>

        <TenantDangerZone />
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

// Archiving and moving a tenant are organization-owner actions, the same ones
// offered from the organization's Tenants list.
const TenantDangerZone: React.FC = () => {
  const { tenant, organizationId } = useTenantDetails();
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { isControlPlaneEnabled } = useControlPlane();
  const { organizations } = useOrganizations();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [showArchiveModal, setShowArchiveModal] = useState(false);
  const [showTransferModal, setShowTransferModal] = useState(false);

  const organization = organizations.find(
    (o) => o.metadata.id === organizationId,
  );

  if (!organizationId || !organization?.isOwner) {
    return null;
  }

  const tenantName = tenant?.name || tenantId;

  return (
    <div className="mt-24 space-y-4">
      <div>
        <h3 className="text-base font-semibold text-destructive">
          Danger Zone
        </h3>
        <p className="mt-1 text-sm text-muted-foreground">
          These actions affect everyone with access to this tenant.
        </p>
      </div>
      <Separator />
      <div className="divide-y divide-border">
        {isControlPlaneEnabled && (
          <SettingRow
            label="Move Tenant"
            description="Move this tenant to another organization you own. Its members are added to that organization."
          >
            <Button
              variant="outline"
              className="shrink-0"
              onClick={() => setShowTransferModal(true)}
              leftIcon={<ArrowsRightLeftIcon className="size-4" />}
            >
              Move
            </Button>
          </SettingRow>
        )}
        <SettingRow
          label="Archive Tenant"
          description="The tenant is kept for 30 days before being permanently deleted."
        >
          <Button
            variant="destructive"
            className="shrink-0"
            onClick={() => setShowArchiveModal(true)}
            leftIcon={<TrashIcon className="size-4" />}
          >
            Archive
          </Button>
        </SettingRow>
      </div>

      {isControlPlaneEnabled && (
        <TransferTenantModal
          open={showTransferModal}
          onOpenChange={setShowTransferModal}
          organizationId={organizationId}
          organizationName={organization.name}
          tenantId={tenantId}
          tenantName={tenantName}
          ownedDestinationOrganizations={organizations.filter(
            (o) => o.isOwner && o.metadata.id !== organizationId,
          )}
          onSuccess={() =>
            queryClient.invalidateQueries({ queryKey: ['user-universe'] })
          }
        />
      )}

      <DeleteTenantModal
        open={showArchiveModal}
        onOpenChange={setShowArchiveModal}
        tenant={{ id: tenantId, status: TenantStatusType.ACTIVE }}
        tenantName={tenantName}
        organizationName={organization.name}
        onSuccess={async () => {
          await queryClient.invalidateQueries({ queryKey: ['user-universe'] });
          navigate({ to: appRoutes.authenticatedRoute.to });
        }}
      />
    </div>
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
  const canWrite = useCanWrite();
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
        disabled={!canWrite}
      />
      {canWrite &&
        changed &&
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

const OnboardingSettingRow: React.FC = () => {
  const { tenantId } = useCurrentTenantId();
  const navigate = useNavigate();
  // The same tenant-scoped key and defaults the Overview onboarding reads,
  // so restarting here is indistinguishable from a first visit there.
  const [, setStoredOnboarding] = useLocalStorageState<unknown>(
    onboardingStorageKey(tenantId),
    null,
  );

  return (
    <SettingRow
      label="Onboarding"
      description="Restart the onboarding guide. This clears onboarding progress for this tenant in this browser."
    >
      <Button
        variant="outline"
        size="sm"
        onClick={() => {
          setStoredOnboarding(defaultOnboardingState());
          navigate({
            to: '/tenants/$tenant/onboarding',
            params: { tenant: tenantId },
          });
        }}
      >
        Restart onboarding
      </Button>
    </SettingRow>
  );
};
