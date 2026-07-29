import { Notification } from './types';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { globalEmitter } from '@/lib/global-emitter';
import { useMemo } from 'react';

export const useInviteNotifications = () => {
  const { pendingInvitesQuery, isLoading } = usePendingInvites();

  const notifications: Notification[] = useMemo(() => {
    const data = pendingInvitesQuery.data;
    if (!data || data.inviteCount === 0) {
      return [];
    }

    const count = data.inviteCount;
    const mostRecent = [...data.tenantInvites, ...data.organizationInvites]
      .map((inv) => inv.metadata.createdAt as string | undefined)
      .filter((ts): ts is string => Boolean(ts))
      .reduce((max, ts) => (ts > max ? ts : max), '');

    return [
      {
        color: 'green',
        shortTitle: count === 1 ? 'Invite' : 'Invites',
        title:
          count === 1
            ? data.tenantInvites[0]
              ? `Invite to join ${data.tenantInvites[0].tenantName ?? 'a tenant'}`
              : 'Organization invite'
            : `${count} pending invites`,
        message: `You have ${count} pending invite${count > 1 ? 's' : ''} awaiting your response.`,
        timestamp: mostRecent,
        onClick: () => globalEmitter.emit('open-invite-modal', {}),
      },
    ];
  }, [pendingInvitesQuery.data]);

  return {
    notifications,
    isLoading,
  };
};
