import { TenantInvite } from '@/lib/api';
import useUser from './use-user';
import { useMemo } from 'react';

type Alert = {
  id: string;
  createdAt: Date;
  invite: TenantInvite;
};

interface AlerterState {
  alerts: Alert[];
}

export default function useAlerter(): AlerterState {
  const { invites } = useUser();

  const alerts = useMemo(() => {
    const alerts = [
      ...invites.list.map(
        (invite) =>
          ({
            id: invite.metadata.id,
            createdAt: new Date(invite.metadata.createdAt),
            invite,
          }) as Alert,
      ),
    ];

    return alerts.sort((a, b) => {
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });
  }, [invites]);

  return {
    alerts,
  };
}
