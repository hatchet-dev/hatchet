import { Combobox } from '@/components/v1/molecules/combobox/combobox';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import useControlPlane from '@/hooks/use-control-plane';
import {
  OrganizationInviteStatus,
  OrganizationMemberRoleType,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { OrganizationInviteTenantRole } from '@/lib/api/generated/control-plane/data-contracts';
import {
  OrganizationInviteCreateRequest,
  useOrganizationApi,
} from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import {
  ExclamationTriangleIcon,
  UserPlusIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState, useEffect, useMemo } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { z } from 'zod';

// API max for tenants / userGroupIds on an invite
const MAX_INVITE_GRANTS = 100;

// One row per tenant, so a duplicate tenantId (a 400 on the server) is
// structurally impossible.
type SelectedTenant = {
  tenantId: string;
  tenantRole: OrganizationInviteTenantRole;
};

const schema = z.object({
  email: z.string().email('Invalid email address'),
  role: z.nativeEnum(OrganizationMemberRoleType),
});

type OrganizationInviteMemberModalProps = {
  organizationId: string;
  organizationName: string;
  onClose: () => void;
  onCreated: (invite: OrganizationInviteCreateRequest) => void;
};

export const OrganizationInviteMemberModal = ({
  organizationId,
  organizationName,
  onClose,
  onCreated,
}: OrganizationInviteMemberModalProps) => {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [selectedTenants, setSelectedTenants] = useState<SelectedTenant[]>([]);
  const [selectedUserGroupIds, setSelectedUserGroupIds] = useState<string[]>(
    [],
  );
  const { isControlPlaneEnabled } = useControlPlane();

  const { handleApiError } = useApiError({
    setFieldErrors,
  });

  const {
    register,
    handleSubmit,
    reset,
    control,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: '',
      role: isControlPlaneEnabled
        ? OrganizationMemberRoleType.MEMBER
        : OrganizationMemberRoleType.OWNER,
    },
  });

  const selectedRole = watch('role');
  const emailValue = watch('email');

  const queryClient = useQueryClient();
  const orgApi = useOrganizationApi();

  // The modal only receives the org id/name, so fetch the org's tenants here.
  const organizationQuery = useQuery({
    ...orgApi.organizationGetQuery(organizationId),
    enabled: !!organizationId && isControlPlaneEnabled,
  });

  const organizationInvitesQuery = useQuery({
    ...orgApi.organizationInviteListQuery(organizationId),
    enabled: !!organizationId && isControlPlaneEnabled,
  });

  // User groups are a control-plane-only concept.
  const userGroupsQuery = useQuery({
    ...orgApi.userGroupsListQuery(organizationId),
    enabled: !!organizationId && isControlPlaneEnabled,
  });

  const tenantOptions = useMemo(
    () =>
      (organizationQuery.data?.tenants ?? [])
        .filter((tenant) => tenant.status !== TenantStatusType.ARCHIVED)
        .map((tenant) => ({
          label: tenant.name ?? tenant.slug ?? tenant.id,
          value: tenant.id,
        })),
    [organizationQuery.data?.tenants],
  );

  const tenantLabelById = useMemo(
    () => new Map(tenantOptions.map((option) => [option.value, option.label])),
    [tenantOptions],
  );

  // Include the group's tenant role in the label since it determines the
  // access level the group confers.
  const userGroupOptions = useMemo(
    () =>
      (userGroupsQuery.data ?? []).map((group) => ({
        label: `${group.name} · ${group.role}`,
        value: group.metadata.id,
      })),
    [userGroupsQuery.data],
  );

  const userGroupLabelById = useMemo(
    () =>
      new Map(userGroupOptions.map((option) => [option.value, option.label])),
    [userGroupOptions],
  );

  // Re-inviting an email with an existing pending invite is a silent no-op on
  // the server — new tenant/group grants would be ignored. Warn (and proceed).
  const hasPendingInviteForEmail = useMemo(() => {
    const email = emailValue?.trim().toLowerCase();
    if (!email) {
      return false;
    }
    return (organizationInvitesQuery.data?.rows ?? []).some(
      (invite) =>
        invite.status === OrganizationInviteStatus.PENDING &&
        invite.inviteeEmail.toLowerCase() === email,
    );
  }, [emailValue, organizationInvitesQuery.data?.rows]);

  const showTenantPicker =
    isControlPlaneEnabled && selectedRole === OrganizationMemberRoleType.MEMBER;
  // Render nothing (rather than an empty picker) when the org has no groups.
  const showUserGroupPicker = showTenantPicker && userGroupOptions.length > 0;

  const orgInviteCreate =
    orgApi.organizationInviteCreateMutation(organizationId);
  const inviteMemberMutation = useMutation({
    ...orgInviteCreate,
    mutationFn: async (data: {
      email: string;
      role: OrganizationMemberRoleType;
    }) => {
      // Never send tenants/userGroupIds for OWNER invites — the server
      // rejects them (owners automatically get access to all tenants).
      const grantsAllowed =
        isControlPlaneEnabled &&
        data.role === OrganizationMemberRoleType.MEMBER;
      const request: OrganizationInviteCreateRequest = {
        inviteeEmail: data.email,
        role: data.role,
        ...(grantsAllowed && selectedTenants.length > 0
          ? {
              tenants: selectedTenants.map(({ tenantId, tenantRole }) => ({
                tenantId,
                tenantRole,
              })),
            }
          : {}),
        ...(grantsAllowed && selectedUserGroupIds.length > 0
          ? { userGroupIds: selectedUserGroupIds }
          : {}),
      };
      await orgInviteCreate.mutationFn(request);
      return request;
    },
    onSuccess: (request) => {
      queryClient.invalidateQueries({
        queryKey: ['organization-invites:list', organizationId],
      });
      reset();
      setSelectedTenants([]);
      setSelectedUserGroupIds([]);
      onCreated(request);
      onClose();
    },
    onError: handleApiError,
  });

  const emailError = errors.email?.message?.toString() || fieldErrors?.email;

  useEffect(() => {
    reset();
    setFieldErrors({});
    setSelectedTenants([]);
    setSelectedUserGroupIds([]);
  }, [reset]);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlusIcon className="h-5 w-5" />
            Invite Member
          </DialogTitle>
          <DialogDescription>
            Invite a new member to {organizationName}
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit((data) => inviteMemberMutation.mutate(data))}
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label htmlFor="email">Email Address</Label>
            <Input
              {...register('email')}
              id="email"
              type="email"
              placeholder="name@example.com"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              disabled={inviteMemberMutation.isPending}
            />
            {emailError && (
              <div className="text-sm text-red-500">{emailError}</div>
            )}
            <p className="text-sm text-muted-foreground">
              The user will receive an email invitation to join this
              organization.
            </p>
            {isControlPlaneEnabled && hasPendingInviteForEmail && (
              <div className="flex items-start gap-2 rounded-md border border-yellow-500/50 bg-yellow-500/10 px-3 py-2 text-sm">
                <ExclamationTriangleIcon className="mt-0.5 size-4 shrink-0 text-yellow-500" />
                <p className="text-muted-foreground">
                  An invite for this email is already pending — cancel it first
                  to change its role, tenant access, or groups.
                </p>
              </div>
            )}
          </div>

          {isControlPlaneEnabled && (
            <div className="space-y-2">
              <Label htmlFor="role">Role</Label>
              <Controller
                control={control}
                name="role"
                render={({ field }) => (
                  <Select
                    onValueChange={(value) => {
                      field.onChange(value);
                      if (value === OrganizationMemberRoleType.OWNER) {
                        setSelectedTenants([]);
                        setSelectedUserGroupIds([]);
                      }
                    }}
                    value={field.value}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select role..." />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={OrganizationMemberRoleType.MEMBER}>
                        Member
                      </SelectItem>
                      <SelectItem value={OrganizationMemberRoleType.OWNER}>
                        Owner
                      </SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
              <p className="text-sm text-muted-foreground">
                Members access tenants based on their tags or can be added
                directly. Owners have full access to all tenants.
              </p>
            </div>
          )}

          {showTenantPicker && (
            <div className="space-y-2">
              <Label>Tenant access (optional)</Label>
              <div>
                <Combobox
                  title="Tenants"
                  type={ToolbarType.Checkbox}
                  options={tenantOptions}
                  values={selectedTenants.map((tenant) => tenant.tenantId)}
                  setValues={(values) => {
                    // Preserve roles already chosen for tenants that stay
                    // selected; new tenants default to MEMBER.
                    const ids = values.slice(0, MAX_INVITE_GRANTS);
                    setSelectedTenants((prev) => {
                      const prevById = new Map(
                        prev.map((tenant) => [tenant.tenantId, tenant]),
                      );
                      return ids.map(
                        (tenantId) =>
                          prevById.get(tenantId) ?? {
                            tenantId,
                            tenantRole: OrganizationInviteTenantRole.MEMBER,
                          },
                      );
                    });
                  }}
                  emptyMessage="No tenants found."
                />
              </div>
              {selectedTenants.length > 0 && (
                <SimpleTable
                  columns={[
                    {
                      columnLabel: 'Tenant',
                      cellRenderer: (row: SelectedTenant) =>
                        tenantLabelById.get(row.tenantId) ?? row.tenantId,
                    },
                    {
                      columnLabel: 'Role',
                      cellRenderer: (row: SelectedTenant) => (
                        <Select
                          value={row.tenantRole}
                          onValueChange={(value) =>
                            setSelectedTenants((prev) =>
                              prev.map((tenant) =>
                                tenant.tenantId === row.tenantId
                                  ? {
                                      ...tenant,
                                      tenantRole:
                                        value as OrganizationInviteTenantRole,
                                    }
                                  : tenant,
                              ),
                            )
                          }
                          disabled={inviteMemberMutation.isPending}
                        >
                          <SelectTrigger
                            className="h-8 w-32"
                            aria-label={`Role for ${tenantLabelById.get(row.tenantId) ?? row.tenantId}`}
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem
                              value={OrganizationInviteTenantRole.MEMBER}
                            >
                              Member
                            </SelectItem>
                            <SelectItem
                              value={OrganizationInviteTenantRole.ADMIN}
                            >
                              Admin
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      ),
                    },
                    {
                      columnLabel: '',
                      cellRenderer: (row: SelectedTenant) => (
                        <button
                          type="button"
                          onClick={() =>
                            setSelectedTenants((prev) =>
                              prev.filter(
                                (tenant) => tenant.tenantId !== row.tenantId,
                              ),
                            )
                          }
                          disabled={inviteMemberMutation.isPending}
                          className="rounded p-1 text-muted-foreground hover:text-destructive disabled:opacity-50"
                          aria-label={`Remove ${tenantLabelById.get(row.tenantId) ?? row.tenantId}`}
                        >
                          <XMarkIcon className="size-4" />
                        </button>
                      ),
                    },
                  ]}
                  data={selectedTenants}
                  rowKey={(row) => row.tenantId}
                />
              )}
              <p className="text-sm text-muted-foreground">
                The invitee will be added to these tenants when they accept.
              </p>
            </div>
          )}

          {showUserGroupPicker && (
            <div className="space-y-2">
              <Label>User groups (optional)</Label>
              <div>
                <Combobox
                  title="User groups"
                  type={ToolbarType.Checkbox}
                  options={userGroupOptions}
                  values={selectedUserGroupIds}
                  setValues={(values) =>
                    setSelectedUserGroupIds(values.slice(0, MAX_INVITE_GRANTS))
                  }
                  emptyMessage="No user groups found."
                />
              </div>
              {selectedUserGroupIds.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {selectedUserGroupIds.map((userGroupId) => (
                    <span
                      key={userGroupId}
                      className="inline-flex items-center gap-1.5 rounded-md border bg-secondary px-3 py-1 text-sm text-secondary-foreground"
                    >
                      {userGroupLabelById.get(userGroupId) ?? userGroupId}
                      <button
                        type="button"
                        onClick={() =>
                          setSelectedUserGroupIds(
                            selectedUserGroupIds.filter(
                              (id) => id !== userGroupId,
                            ),
                          )
                        }
                        disabled={inviteMemberMutation.isPending}
                        className="ml-0.5 rounded hover:text-destructive disabled:opacity-50"
                        aria-label={`Remove ${userGroupLabelById.get(userGroupId) ?? userGroupId}`}
                      >
                        <XMarkIcon className="size-3.5" />
                      </button>
                    </span>
                  ))}
                </div>
              )}
              <p className="text-sm text-muted-foreground">
                Group membership grants tenant access dynamically by tags.
              </p>
            </div>
          )}

          <div className="flex items-center justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={inviteMemberMutation.isPending}>
              {inviteMemberMutation.isPending
                ? 'Sending...'
                : 'Send Invitation'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
