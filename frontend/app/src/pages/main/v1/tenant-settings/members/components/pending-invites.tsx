import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { InlineError } from '@/components/v1/ui/inline-error';
import { TenantInvite } from '@/lib/api';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import {
  MemberEmail,
  RoleBadge,
} from '@/pages/main/v1/tenant-settings/components/member-primitives';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

export function PendingInvitesSection({
  invites,
  tenantId,
  canManage,
  onInviteRevoked,
}: {
  invites: TenantInvite[];
  tenantId: string;
  canManage: boolean;
  onInviteRevoked: () => void;
}) {
  const [inviteToRevoke, setInviteToRevoke] = useState<TenantInvite | null>(
    null,
  );

  if (invites.length === 0) {
    return null;
  }

  const columns = [
    {
      columnLabel: 'Email',
      cellRenderer: (invite: TenantInvite) => (
        <MemberEmail email={invite.email} />
      ),
    },
    {
      columnLabel: 'Role',
      cellRenderer: (invite: TenantInvite) => <RoleBadge role={invite.role} />,
    },
    {
      columnLabel: 'Created',
      cellRenderer: (invite: TenantInvite) => (
        <RelativeDate date={invite.metadata.createdAt} />
      ),
    },
    {
      columnLabel: 'Expires',
      cellRenderer: (invite: TenantInvite) => (
        <RelativeDate date={invite.expires} />
      ),
    },
    ...(canManage
      ? [
          {
            columnLabel: 'Actions',
            cellRenderer: (invite: TenantInvite) => (
              <TableRowActions
                row={invite}
                actions={[
                  {
                    label: 'Revoke invite',
                    onClick: () => setInviteToRevoke(invite),
                  },
                ]}
              />
            ),
          },
        ]
      : []),
  ];

  return (
    <div className="mt-6 space-y-2">
      <h3 className="text-base font-semibold">Pending Invites</h3>
      <SimpleTable
        columns={columns}
        data={invites}
        rowKey={(invite) => invite.metadata.id}
      />

      {inviteToRevoke && (
        <RevokeInviteDialog
          tenantId={tenantId}
          invite={inviteToRevoke}
          onClose={() => setInviteToRevoke(null)}
          onSuccess={() => {
            setInviteToRevoke(null);
            onInviteRevoked();
          }}
        />
      )}
    </div>
  );
}

function RevokeInviteDialog({
  tenantId,
  invite,
  onClose,
  onSuccess,
}: {
  tenantId: string;
  invite: TenantInvite;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });
  const { tenantInviteDeleteMutation } = useTenantApi();
  const deleteInvite = tenantInviteDeleteMutation(tenantId, invite.metadata.id);
  const revokeMutation = useMutation({
    ...deleteInvite,
    onSuccess,
    // Keep the dialog open so the inline error is visible.
    onError: handleApiError,
  });

  return (
    <ConfirmDialog
      isOpen
      title="Revoke invite"
      description={
        <div className="space-y-3">
          <p>
            Revoke the pending invite for <strong>{invite.email}</strong>? They
            will no longer be able to accept it.
          </p>
          <InlineError errors={formErrors} />
        </div>
      }
      submitLabel="Revoke invite"
      submitVariant="destructive"
      isLoading={revokeMutation.isPending}
      onSubmit={() => {
        setFormErrors([]);
        revokeMutation.mutate();
      }}
      onCancel={onClose}
    />
  );
}
