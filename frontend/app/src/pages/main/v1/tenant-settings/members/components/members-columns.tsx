import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { TenantMember, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { UserContextType } from '@/lib/outlet';
import { useOutletContext } from '@/lib/router-helpers';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import useCloud from '@/pages/auth/hooks/use-cloud';
import queryClient from '@/query-client';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

// Component for handling member actions
export function MemberActions({
  member,
  onChangePasswordClick,
  onEditRoleClick,
}: {
  member: TenantMember;
  onChangePasswordClick: (member: TenantMember) => void;
  onEditRoleClick: (member: TenantMember) => void;
}) {
  const { user } = useOutletContext<UserContextType>();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const { handleApiError } = useApiError({});
  const { tenantId } = useCurrentTenantId();
  const { meta } = useApiMeta();
  const { isCloudEnabled } = useCloud();

  const deleteMemberMutation = useMutation({
    mutationKey: ['tenant-member:delete', tenantId],
    mutationFn: async (data: { memberId: string }) => {
      await api.tenantMemberDelete(tenantId, data.memberId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.members.list(tenantId).queryKey,
      });
    },
    onError: handleApiError,
  });

  const isOwnerRole = member.role === 'OWNER';

  const canDeleteMember =
    member.user.email !== user?.email &&
    meta?.allowInvites &&
    !(isCloudEnabled && isOwnerRole); // Hide delete option for OWNER in cloud mode

  const canChangePassword =
    member.user.email === user?.email && meta?.allowChangePassword;

  const canEditRole = member.user.email !== user?.email;

  return (
    <>
      <TableRowActions
        row={member}
        actions={[
          ...(canEditRole
            ? [
                {
                  label: 'Edit role',
                  onClick: () => onEditRoleClick(member),
                },
              ]
            : []),
          ...(canChangePassword
            ? [
                {
                  label: 'Change password',
                  onClick: () => onChangePasswordClick(member),
                },
              ]
            : []),
          ...(canDeleteMember
            ? [
                {
                  label: 'Remove from tenant',
                  onClick: () => setShowDeleteDialog(true),
                },
              ]
            : []),
        ]}
      />
      <ConfirmDialog
        isOpen={showDeleteDialog}
        title="Remove member"
        description={`Are you sure you want to remove ${member.user.name} from this tenant?`}
        submitLabel="Remove"
        onSubmit={() => {
          deleteMemberMutation.mutate({
            memberId: member.metadata.id,
          });
        }}
        onCancel={() => setShowDeleteDialog(false)}
        isLoading={deleteMemberMutation.isPending}
      />
    </>
  );
}
