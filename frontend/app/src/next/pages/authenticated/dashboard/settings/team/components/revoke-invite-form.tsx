import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/next/components/ui/alert-dialog';
import { TenantInvite } from '@/lib/api/generated/data-contracts';
import useMembers from '@/next/hooks/use-members';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import api from '@/lib/api';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';
import { Button } from '@/next/components/ui/button';

interface RevokeInviteFormProps {
  invite: TenantInvite;
  close: () => void;
}

export function RevokeInviteForm({ invite, close }: RevokeInviteFormProps) {
  const { tenantId } = useCurrentTenantId();
  const { refetchInvites } = useMembers();
  const [error, setError] = useState<string | null>(null);

  const revokeMutation = useMutation({
    mutationFn: async () => {
      await api.tenantInviteDelete(tenantId, invite.metadata.id);
    },
    onSuccess: () => {
      refetchInvites();
      close();
    },
    onError: (err: any) => {
      setError('Failed to revoke invite. Please try again.');
      console.error('Error revoking invite:', err);
    },
  });

  return (
    <AlertDialog open onOpenChange={close}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke Invite</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to revoke the invitation for{' '}
            <strong>{invite.email}</strong>? This action cannot be undone.
          </AlertDialogDescription>
          {error && <p className="text-destructive mt-2">{error}</p>}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button variant="outline" onClick={close}>
              Cancel
            </Button>
          </AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button
              variant="destructive"
              onClick={(e) => {
                e.preventDefault();
                revokeMutation.mutate();
              }}
              loading={revokeMutation.isPending}
            >
              Revoke Invite
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
