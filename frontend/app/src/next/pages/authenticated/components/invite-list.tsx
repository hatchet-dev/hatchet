import { Button } from '@/next/components/ui/button';
import { TenantInvite, TenantMemberRole } from '@/lib/api';
import { useState } from 'react';
import { formatDistance } from 'date-fns';
import api from '@/lib/api';
import { useMutation } from '@tanstack/react-query';
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import { CheckCircle, XCircle, Clock } from 'lucide-react';
import { Badge } from '@/next/components/ui/badge';

interface InviteListProps {
  invites: TenantInvite[];
  onInviteAccepted?: () => void;
  onInviteRejected?: () => void;
}

export function InviteList({
  invites,
  onInviteAccepted,
  onInviteRejected,
}: InviteListProps) {
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const acceptInviteMutation = useMutation({
    mutationKey: ['tenant-invite:accept'],
    mutationFn: async (inviteId: string) => {
      await api.tenantInviteAccept({ invite: inviteId });
    },
    onSuccess: () => {
      setSuccess('Invite accepted successfully');
      if (onInviteAccepted) {
        onInviteAccepted();
      }
    },
    onError: (error: any) => {
      setError(error?.response?.data?.message || 'Failed to accept invite');
    },
  });

  const rejectInviteMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (inviteId: string) => {
      await api.tenantInviteReject({ invite: inviteId });
    },
    onSuccess: () => {
      setSuccess('Invite rejected successfully');
      if (onInviteRejected) {
        onInviteRejected();
      }
    },
    onError: (error: any) => {
      setError(error?.response?.data?.message || 'Failed to reject invite');
    },
  });

  if (invites.length === 0) {
    return null;
  }

  // Format role for display
  const formatRole = (role: TenantMemberRole) => {
    switch (role) {
      case TenantMemberRole.OWNER:
        return 'Owner';
      case TenantMemberRole.ADMIN:
        return 'Admin';
      case TenantMemberRole.MEMBER:
        return 'Member';
      default:
        return role;
    }
  };

  // Get badge variant based on role
  const getRoleBadgeVariant = (role: TenantMemberRole) => {
    switch (role) {
      case TenantMemberRole.OWNER:
        return 'default';
      case TenantMemberRole.ADMIN:
        return 'secondary';
      case TenantMemberRole.MEMBER:
        return 'outline';
      default:
        return 'outline';
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-xl font-semibold mb-2">Pending Invites</h2>

      {error && (
        <Alert variant="destructive" className="mb-4">
          <XCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {success && (
        <Alert className="mb-4">
          <CheckCircle className="h-4 w-4" />
          <AlertTitle>Success</AlertTitle>
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}

      {invites.map((invite) => (
        <div
          key={invite.metadata.id}
          className="border rounded-md p-4 bg-card hover:bg-accent/5 transition-colors"
        >
          <div className="flex items-center justify-between">
            <div className="flex flex-col">
              <div className="flex items-center gap-2">
                <h3 className="font-medium">{invite.tenantName || 'Team'}</h3>
                <Badge variant={getRoleBadgeVariant(invite.role)}>
                  {formatRole(invite.role)}
                </Badge>
              </div>

              <div className="flex items-center text-sm text-muted-foreground mt-1">
                <Clock className="h-3.5 w-3.5 mr-1 inline" />
                Expires{' '}
                {formatDistance(new Date(invite.expires), new Date(), {
                  addSuffix: true,
                })}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => rejectInviteMutation.mutate(invite.metadata.id)}
                disabled={
                  rejectInviteMutation.isPending ||
                  acceptInviteMutation.isPending
                }
              >
                Decline
              </Button>
              <Button
                size="sm"
                onClick={() => acceptInviteMutation.mutate(invite.metadata.id)}
                disabled={
                  rejectInviteMutation.isPending ||
                  acceptInviteMutation.isPending
                }
              >
                Accept
              </Button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
