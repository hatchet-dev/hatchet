import { useInviteNotifications } from './invites';
import { useOnboardingNotifications } from './onboarding';
import { useResourceLimitNotifications } from './resource-limits';
import { useMemo } from 'react';

export type { Notification, NotificationColor } from './types';

export const useNotifications = () => {
  const resourceLimits = useResourceLimitNotifications();
  const invites = useInviteNotifications();
  const onboarding = useOnboardingNotifications();

  const notifications = useMemo(
    () =>
      [
        ...resourceLimits.notifications,
        ...invites.notifications,
        ...onboarding.notifications,
      ].sort((a, b) => (b.timestamp > a.timestamp ? 1 : -1)),
    [
      resourceLimits.notifications,
      invites.notifications,
      onboarding.notifications,
    ],
  );

  return {
    notifications,
    isLoading:
      resourceLimits.isLoading || invites.isLoading || onboarding.isLoading,
  };
};
