import MainNav from '@/components/molecules/nav-bar/nav-bar';
import { Outlet } from 'react-router-dom';
import api, { queries, TenantVersion } from '@/lib/api';
import { Loading } from '@/components/ui/loading.tsx';
import { useQuery } from '@tanstack/react-query';
import SupportChat from '@/components/molecules/support-chat';
import AnalyticsProvider from '@/components/molecules/analytics-provider';
import { useState, useEffect } from 'react';
import { useContextFromParent } from '@/lib/outlet';
import { useTenant } from '@/lib/atoms';

export default function Authenticated() {
  const [hasHasBanner, setHasBanner] = useState(false);

  const { tenant } = useTenant();

  const userQuery = useQuery({
    queryKey: ['user:get:current'],
    retry: false,
    queryFn: async () => {
      const res = await api.userGetCurrent();
      return res.data;
    },
  });

  const invitesQuery = useQuery({
    queryKey: ['user:list-tenant-invites'],
    retry: false,
    queryFn: async () => {
      const res = await api.userListTenantInvites();
      return res.data.rows || [];
    },
  });

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
    retry: false,
  });

  const ctx = useContextFromParent({
    user: userQuery.data,
    memberships: listMembershipsQuery.data?.rows,
  });

  useEffect(() => {
    const currentUrl = window.location.pathname;

    if (
      userQuery.data &&
      !userQuery.data.emailVerified &&
      !currentUrl.includes('/onboarding/verify-email')
    ) {
      window.location.href = '/onboarding/verify-email';
      return;
    }

    if (
      invitesQuery.data?.length &&
      invitesQuery.data.length > 0 &&
      !currentUrl.includes('/onboarding/invites')
    ) {
      window.location.href = '/onboarding/invites';
      return;
    }

    if (
      listMembershipsQuery.data?.rows?.length === 0 &&
      !currentUrl.includes('/onboarding')
    ) {
      window.location.href = '/onboarding/create-tenant';
      return;
    }

    if (
      tenant?.version === TenantVersion.V0 &&
      currentUrl.startsWith('/tenants')
    ) {
      window.location.href = `/workflow-runs?tenant=${tenant.metadata.id}`;
      return;
    }
  }, [
    tenant?.metadata.id,
    userQuery.data,
    invitesQuery.data,
    listMembershipsQuery.data,
    tenant?.version,
  ]);

  if (
    userQuery.isLoading ||
    invitesQuery.isLoading ||
    listMembershipsQuery.isLoading
  ) {
    return <Loading />;
  }

  if (userQuery.error) {
    const currentUrl = window.location.pathname;
    if (
      !currentUrl.includes('/auth/login') &&
      !currentUrl.includes('/auth/register')
    ) {
      window.location.href = '/auth/login';
      return null;
    }
  }

  if (!userQuery.data || !listMembershipsQuery.data?.rows) {
    return <Loading />;
  }

  return (
    <AnalyticsProvider user={userQuery.data}>
      <SupportChat user={userQuery.data}>
        <div className="flex flex-row flex-1 w-full h-full">
          <MainNav user={userQuery.data} setHasBanner={setHasBanner} />
          <div
            className={`${hasHasBanner ? 'pt-28' : 'pt-16'} flex-grow overflow-y-auto overflow-x-hidden`}
          >
            <Outlet context={ctx} />
          </div>
        </div>
      </SupportChat>
    </AnalyticsProvider>
  );
}
