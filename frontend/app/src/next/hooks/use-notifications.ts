import { TenantInvite } from '@/lib/api';
import useUser from './use-user';
import { useMemo } from 'react';

type Notification = {
  id: string;
  createdAt: Date;
  invite: TenantInvite;
};

interface NotificationState {
  notifications: Notification[];
}

export default function useNotifications(): NotificationState {
  const { invites } = useUser();

  const notifications = useMemo(() => {
    const alerts = [
      ...invites.list.map(
        (invite) =>
          ({
            id: invite.metadata.id,
            createdAt: new Date(invite.metadata.createdAt),
            invite,
          }) as Notification,
      ),
    ];

    return alerts.sort((a, b) => {
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });
  }, [invites]);

  return {
    notifications,
  };
}
