import { useBillingUsageNotifications } from './billing-usage';
import {
  isLimitNotificationDismissed,
  useDismissedLimitNotifications,
} from './dismissed';
import { useInviteNotifications } from './invites';
import { useOnboardingNotifications } from './onboarding';
import { useResourceLimitNotifications } from './resource-limits';
import { useMemo } from 'react';

export type { Notification, NotificationColor } from './types';

export const useNotifications = () => {
  const resourceLimits = useResourceLimitNotifications();
  const billingUsage = useBillingUsageNotifications();
  const invites = useInviteNotifications();
  const onboarding = useOnboardingNotifications();
  const { dismissed, dismiss } = useDismissedLimitNotifications();

  const notifications = useMemo(
    () =>
      [
        ...resourceLimits.notifications,
        ...billingUsage.notifications,
        ...invites.notifications,
        ...onboarding.notifications,
      ]
        .filter((n) => !isLimitNotificationDismissed(dismissed, n.dismissKey))
        .sort((a, b) => (b.timestamp > a.timestamp ? 1 : -1)),
    [
      resourceLimits.notifications,
      billingUsage.notifications,
      invites.notifications,
      onboarding.notifications,
      dismissed,
    ],
  );

  return {
    notifications,
    dismiss,
    isLoading:
      resourceLimits.isLoading ||
      billingUsage.isLoading ||
      invites.isLoading ||
      onboarding.isLoading,
  };
};
