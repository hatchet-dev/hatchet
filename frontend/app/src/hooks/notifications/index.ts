import { useInviteNotifications } from './invites';
import { useResourceLimitNotifications } from './resource-limits';
import { useMemo } from 'react';

export type { Notification, NotificationColor } from './types';

export const useNotifications = () => {
  const resourceLimits = useResourceLimitNotifications();
  const invites = useInviteNotifications();

  const notifications = useMemo(
    () =>
      [...resourceLimits.notifications, ...invites.notifications].sort(
        (a, b) => (b.timestamp > a.timestamp ? 1 : -1),
      ),
    [resourceLimits.notifications, invites.notifications],
  );

  return {
    notifications,
    isLoading: resourceLimits.isLoading || invites.isLoading,
  };
};
