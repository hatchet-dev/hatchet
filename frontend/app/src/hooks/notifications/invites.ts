import { Notification } from './types';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { appRoutes } from '@/router';
import { useMemo } from 'react';

const startsWithVowelRegex = /^[aeiou]/i;

const getArticle = (word: string) =>
  startsWithVowelRegex.test(word) ? 'an' : 'a';

export const useInviteNotifications = () => {
  const { pendingInvitesQuery, isLoading } = usePendingInvites();

  const notifications: Notification[] = useMemo(() => {
    const data = pendingInvitesQuery.data;
    if (!data || data.inviteCount === 0) {
      return [];
    }

    return [
      ...data.tenantInvites.map(
        (invite): Notification => ({
          color: 'green',
          title: `Tenant invite: ${invite.tenantName ?? 'Unknown'}`,
          message: `You're invited to ${invite.tenantName ? `the "${invite.tenantName}"` : 'a'} tenant as ${getArticle(invite.role)} ${invite.role.toLowerCase()}`,
          timestamp: invite.metadata.createdAt,
          url: appRoutes.onboardingInvitesRoute.to,
        }),
      ),
      ...data.organizationInvites.map(
        (invite): Notification => ({
          color: 'green',
          title: 'Organization invite',
          message: `You're invited to be ${getArticle(invite.role)} ${invite.role.toLowerCase()} of an organization`,
          timestamp: invite.metadata.createdAt,
          url: appRoutes.onboardingInvitesRoute.to,
        }),
      ),
    ];
  }, [pendingInvitesQuery.data]);

  return {
    notifications,
    isLoading,
  };
};
