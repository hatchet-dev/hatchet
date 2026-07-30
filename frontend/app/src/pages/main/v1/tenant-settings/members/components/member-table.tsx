import { ChangePasswordDialog } from './change-password-dialog';
import { MemberActions } from './members-columns';
import { UpdateMemberForm } from './update-member-form';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Badge } from '@/components/v1/ui/badge';
import { Dialog } from '@/components/v1/ui/dialog';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import useControlPlane from '@/hooks/use-control-plane';
import { TenantMember, TenantMemberRole } from '@/lib/api';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import {
  MemberEmail,
  RoleBadge,
} from '@/pages/main/v1/tenant-settings/components/member-primitives';
import { useMutation } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useMemo, useState } from 'react';

function MemberSourceBadge({ member }: { member: TenantMember }) {
  const isDirect = member.manually_added;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant="outline">{isDirect ? 'Direct' : 'Tags'}</Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {isDirect
              ? 'Added to this tenant directly, rather than through tag matching.'
              : "Membership derived from matching the organization's tenant tags."}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function MemberTable({
  tenantId,
  members,
  canManage,
  canManageOrganization,
  onMembersChanged,
}: {
  tenantId: string;
  members: TenantMember[];
  canManage: boolean;
  canManageOrganization: boolean;
  onMembersChanged: () => void;
}) {
  const [memberToEdit, setMemberToEdit] = useState<TenantMember | null>(null);
  const [showChangePassword, setShowChangePassword] = useState(false);
  const { isControlPlaneEnabled } = useControlPlane();

  const columns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (member: TenantMember) => (
          <span className="font-medium">{member.user.name}</span>
        ),
      },
      {
        columnLabel: 'Email',
        cellRenderer: (member: TenantMember) => (
          <MemberEmail email={member.user.email} />
        ),
      },
      {
        columnLabel: 'Role',
        cellRenderer: (member: TenantMember) => (
          <RoleBadge role={member.role} />
        ),
      },
      {
        columnLabel: 'Joined',
        cellRenderer: (member: TenantMember) => (
          <RelativeDate date={member.metadata.createdAt} />
        ),
      },
      ...(isControlPlaneEnabled
        ? [
            {
              columnLabel: 'Source',
              cellRenderer: (member: TenantMember) => (
                <MemberSourceBadge member={member} />
              ),
            },
          ]
        : []),
      ...(canManage
        ? [
            {
              columnLabel: 'Actions',
              cellRenderer: (member: TenantMember) => (
                <MemberActions
                  member={member}
                  tenantId={tenantId}
                  onEditRoleClick={
                    // tag-derived memberships can't have their role edited
                    // here; only manually-added members are editable on the
                    // control plane
                    !isControlPlaneEnabled || member.manually_added
                      ? setMemberToEdit
                      : () => {}
                  }
                  onChangePasswordClick={() => setShowChangePassword(true)}
                  onDeleteSuccess={onMembersChanged}
                />
              ),
            },
          ]
        : []),
    ],
    [canManage, isControlPlaneEnabled, onMembersChanged, tenantId],
  );

  return (
    <>
      <SimpleTable
        columns={columns}
        data={members}
        rowKey={(member) => member.metadata.id}
      />

      {showChangePassword && (
        <ChangePasswordDialog onClose={() => setShowChangePassword(false)} />
      )}

      {memberToEdit && (
        <EditMemberRoleDialog
          tenantId={tenantId}
          member={memberToEdit}
          canManageOrganization={canManageOrganization}
          onClose={() => setMemberToEdit(null)}
          onSuccess={() => {
            setMemberToEdit(null);
            onMembersChanged();
          }}
        />
      )}
    </>
  );
}

function EditMemberRoleDialog({
  tenantId,
  member,
  canManageOrganization,
  onClose,
  onSuccess,
}: {
  tenantId: string;
  member: TenantMember;
  canManageOrganization: boolean;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });
  const { tenantMemberUpdateMutation } = useTenantApi();
  const memberUpdate = tenantMemberUpdateMutation(tenantId, member.metadata.id);
  const updateMutation = useMutation({
    ...memberUpdate,
    onSuccess,
    // Keep the dialog open so the inline error is visible.
    onError: (error: AxiosError) => handleApiError(error),
  });

  const handleSubmit = (data: { role: TenantMemberRole }) => {
    setFormErrors([]);
    if (member.role === TenantMemberRole.OWNER && !canManageOrganization) {
      setFormErrors([
        'Owner role management must be done through organization membership.',
      ]);
      return;
    }
    updateMutation.mutate(data);
  };

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <UpdateMemberForm
        isLoading={updateMutation.isPending}
        onSubmit={handleSubmit}
        formErrors={formErrors}
        member={member}
        isControlPlaneEnabled={true}
        canSetOwnerRole={canManageOrganization}
      />
    </Dialog>
  );
}
