import { SettingsPageHeader } from '../components/settings-page-header';
import { AddOIDCGroupMappingDialog } from './components/add-oidc-group-mapping-dialog';
import { MemberTable } from './components/member-table';
import { PendingInvitesSection } from './components/pending-invites';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { OIDCGroupMapping, TenantMemberRole } from '@/lib/api';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { globalEmitter } from '@/lib/global-emitter';
import { useApiError } from '@/lib/hooks';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { RoleBadge } from '@/pages/main/v1/tenant-settings/components/member-primitives';
import { appRoutes } from '@/router';
import { PlusIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useMemo } from 'react';

export default function Members() {
  const { tenantId } = useCurrentTenantId();
  const { membership, organizationId } = useTenantDetails();
  const { isControlPlaneEnabled } = useControlPlane();
  const { organizations } = useOrganizations();
  const {
    tenantMemberListQuery,
    tenantInviteListQuery,
    tenantOIDCGroupMappingDeleteMutation,
  } = useTenantApi();
  const { handleApiError } = useApiError({});
  const { meta } = useApiMeta();
  const oidcEnabled = meta?.auth?.schemes?.includes('oidc') ?? false;

  const canManageOrganization =
    organizations.find((o) => o.metadata.id === organizationId)?.isOwner ??
    false;

  // `membership` is the current user's role in this tenant
  const isTenantAdmin =
    membership === TenantMemberRole.OWNER ||
    membership === TenantMemberRole.ADMIN;

  // Without the control plane, tenant admins/owners can manage members
  // directly (org-based "Add members" isn't available), whether they're on
  // OSS or on cloud.
  const canManageTenantMembers =
    canManageOrganization || (!isControlPlaneEnabled && isTenantAdmin);

  const membersQuery = useQuery(tenantMemberListQuery(tenantId));
  const invitesQuery = useQuery(tenantInviteListQuery(tenantId));
  const deleteOIDCMapping = useMutation({
    ...tenantOIDCGroupMappingDeleteMutation(tenantId),
    onSuccess: () => membersQuery.refetch(),
    onError: handleApiError,
  });

  const members = useMemo(
    () => membersQuery.data?.rows || [],
    [membersQuery.data?.rows],
  );
  const invites = invitesQuery.data?.rows || [];
  const oidcGroupMappings: OIDCGroupMapping[] =
    membersQuery.data && 'oidcGroupMappings' in membersQuery.data
      ? membersQuery.data.oidcGroupMappings || []
      : [];

  const membersForbidden =
    membersQuery.isError &&
    membersQuery.error instanceof AxiosError &&
    [401, 403].includes(membersQuery.error.response?.status ?? 0);

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        {oidcEnabled && (
          <section className="mb-8 space-y-2">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h2 className="text-base font-semibold">OIDC groups</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  Groups that grant access to this tenant on the next sign-in.
                </p>
              </div>
              {canManageTenantMembers && !isControlPlaneEnabled && (
                <AddOIDCGroupMappingDialog
                  tenantId={tenantId}
                  onCreated={() => membersQuery.refetch()}
                />
              )}
            </div>
            <Separator />
            {oidcGroupMappings.length > 0 ? (
              <SimpleTable
                data={oidcGroupMappings}
                columns={[
                  {
                    columnLabel: 'Group',
                    cellRenderer: (mapping) => (
                      <span className="font-medium">{mapping.group}</span>
                    ),
                  },
                  {
                    columnLabel: 'Tenant role',
                    cellRenderer: (mapping) => (
                      <RoleBadge role={mapping.role} />
                    ),
                  },
                  {
                    columnLabel: 'Actions',
                    cellRenderer: (mapping) =>
                      mapping.id && canManageTenantMembers ? (
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          title="Delete OIDC group mapping"
                          aria-label={`Delete ${mapping.group} mapping`}
                          disabled={deleteOIDCMapping.isPending}
                          onClick={() => {
                            if (
                              window.confirm(
                                `Delete the OIDC mapping for ${mapping.group}?`,
                              )
                            ) {
                              deleteOIDCMapping.mutate(mapping.id!);
                            }
                          }}
                        >
                          <TrashIcon className="size-4" />
                        </Button>
                      ) : null,
                  },
                ]}
                rowKey={(mapping) => `${mapping.group}:${mapping.role}`}
              />
            ) : (
              <p className="text-sm text-muted-foreground">
                No OIDC group mappings are configured for this tenant.
              </p>
            )}
          </section>
        )}

        <SettingsPageHeader
          title="Members"
          description={
            <>
              Manage which team members can access this tenant.
              {canManageOrganization && organizationId && (
                <>
                  {' '}
                  Looking to add new team members? Navigate to{' '}
                  <Link
                    to={appRoutes.organizationTeamRoute.to}
                    params={{ organization: organizationId }}
                    className="text-primary underline-offset-4 hover:underline"
                  >
                    team settings
                  </Link>
                  .
                </>
              )}
            </>
          }
        />

        {canManageTenantMembers && (
          <div className="mb-4 flex flex-row items-baseline justify-end">
            <Button
              variant="link"
              className="h-auto p-0"
              leftIcon={<PlusIcon className="size-4" />}
              onClick={() =>
                globalEmitter.emit('create-tenant-invite', {
                  tenantId,
                  organizationId,
                })
              }
            >
              Add members to this tenant
            </Button>
          </div>
        )}

        {membersQuery.isLoading ? (
          <div className="py-4 text-sm text-muted-foreground">
            Loading members...
          </div>
        ) : membersForbidden ? (
          <div className="py-4 text-sm text-muted-foreground">
            You must be a tenant admin or owner to view members.
          </div>
        ) : members.length > 0 ? (
          <MemberTable
            tenantId={tenantId}
            members={members}
            canManage={canManageTenantMembers}
            canManageOrganization={canManageOrganization}
            onMembersChanged={() => membersQuery.refetch()}
          />
        ) : (
          <EmptyState
            title="No members found"
            description="Members with access to this tenant will appear here."
          />
        )}

        <PendingInvitesSection
          invites={invites}
          tenantId={tenantId}
          canManage={canManageTenantMembers}
          onInviteRevoked={() => invitesQuery.refetch()}
        />
      </div>
    </div>
  );
}
