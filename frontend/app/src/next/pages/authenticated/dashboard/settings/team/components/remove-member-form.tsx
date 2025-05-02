import { TenantMember } from '@/lib/api/generated/data-contracts';
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
import { Button } from '@/next/components/ui/button';
import { useMutation } from '@tanstack/react-query';
import api from '@/lib/api';
import useTenant from '@/next/hooks/use-tenant';
import { useState } from 'react';
import useCan from '@/next/hooks/use-can';
import { members } from '@/next/lib/can/features/members.permissions';

interface RemoveMemberFormProps {
  member: TenantMember;
  close: () => void;
}

export function RemoveMemberForm({ member, close }: RemoveMemberFormProps) {
  const { tenant } = useTenant();
  const [error, setError] = useState<string | null>(null);
  const { canWithReason } = useCan();

  const { allowed: canRemove, message } = canWithReason(members.remove(member));

  const mutation = useMutation({
    mutationKey: ['remove-member', tenant?.metadata.id, member.metadata.id],
    mutationFn: async () => {
      if (!tenant?.metadata.id) {
        throw new Error('No tenant selected');
      }
      await api.tenantMemberDelete(tenant.metadata.id, member.metadata.id);
    },
    onSuccess: () => {
      close();
    },
    onError: (err) => {
      console.error('Failed to remove member:', err);
      setError('Failed to remove member. Please try again.');
    },
  });

  if (!canRemove) {
    return (
      <AlertDialog open={!!member} onOpenChange={close}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Permission Denied</AlertDialogTitle>
            <AlertDialogDescription>
              {message || 'You do not have permission to remove this member.'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={close}>Close</AlertDialogCancel>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  }

  return (
    <AlertDialog open={!!member} onOpenChange={close}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Remove member</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to remove {member.user.email} from this
            tenant? This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error && <p className="text-destructive text-sm mt-2">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel onClick={close} disabled={mutation.isPending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button
              variant="destructive"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {mutation.isPending ? 'Removing...' : 'Remove'}
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
