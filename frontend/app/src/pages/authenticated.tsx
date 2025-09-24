import MainNav from '@/components/molecules/nav-bar/nav-bar';
import { Outlet, useNavigate } from 'react-router-dom';
import api, { queries, User } from '@/lib/api';
import { Loading } from '@/components/ui/loading';
import { useMutation, useQuery } from '@tanstack/react-query';
import SupportChat from '@/components/molecules/support-chat';
import AnalyticsProvider from '@/components/molecules/analytics-provider';
import { useEffect } from 'react';
import { useContextFromParent } from '@/lib/outlet';
import { useTenant } from '@/lib/atoms';
import { AxiosError } from 'axios';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { cloudApi } from '@/lib/api/api';

export default function Authenticated() {
  const { tenant } = useTenant();

  const { data: cloudMetadata } = useQuery({
    queryKey: ['metadata'],
    queryFn: async () => {
      const res = await cloudApi.metadataGet();
      return res.data;
    },
  });

  const navigate = useNavigate();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate('/auth/login');
    },
  });

  useInactivityDetection({
    timeoutMs: cloudMetadata?.inactivityLogoutMs || -1,
    onInactive: () => {
      logoutMutation.mutate();
    },
  });

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
    const userQueryError = userQuery.error as AxiosError<User> | null;

    // Skip all redirects for organization pages
    if (currentUrl.startsWith('/organizations')) {
      return;
    }

    if (userQueryError?.status === 401 || userQueryError?.status === 403) {
      window.location.href = '/auth/login';
      return;
    }

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
  }, [
    tenant?.metadata.id,
    userQuery.data,
    invitesQuery.data,
    listMembershipsQuery.data,
    tenant?.version,
    userQuery.error,
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

  if (!userQuery.data) {
    return <Loading />;
  }

  // Allow organization pages even without tenant memberships
  const isOrgPage = window.location.pathname.includes('/organizations');
  if (!isOrgPage && !listMembershipsQuery.data?.rows) {
    return <Loading />;
  }

  return (
    <AnalyticsProvider user={userQuery.data}>
      <SupportChat user={userQuery.data}>
        <div className="flex flex-row flex-1 w-full h-full">
          <MainNav user={userQuery.data} />
          <div className={'pt-16 flex-grow overflow-y-auto overflow-x-hidden'}>
            <Outlet context={ctx} />
          </div>
        </div>
      </SupportChat>
    </AnalyticsProvider>
  );
}
