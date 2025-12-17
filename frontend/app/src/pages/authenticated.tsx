import MainNav from '@/components/molecules/nav-bar/nav-bar';
import {
  useLocation,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router';
import api, { queries, User } from '@/lib/api';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useMutation, useQuery } from '@tanstack/react-query';
import SupportChat from '@/components/molecules/support-chat';
import { PostHogProvider } from '@/providers/posthog';
import { useEffect } from 'react';
import { useContextFromParent } from '@/lib/outlet';
import { AxiosError } from 'axios';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { cloudApi } from '@/lib/api/api';
import { useTenantDetails } from '@/hooks/use-tenant';
import { OutletWithContext } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import { useAtom } from 'jotai';
import { lastTenantAtom } from '@/lib/atoms';

export default function Authenticated() {
  const { tenant } = useTenantDetails();
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);

  const { data: cloudMetadata } = useQuery({
    queryKey: ['metadata'],
    queryFn: async () => {
      const res = await cloudApi.metadataGet();
      return res.data;
    },
  });

  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const matchRoute = useMatchRoute();
  const isAuthPage =
    Boolean(matchRoute({ to: appRoutes.authLoginRoute.to })) ||
    Boolean(matchRoute({ to: appRoutes.authRegisterRoute.to }));
  const isOrganizationsPage = Boolean(
    matchRoute({ to: appRoutes.organizationsRoute.to, fuzzy: true }),
  );
  const isOnboardingVerifyEmailPage = Boolean(
    matchRoute({ to: appRoutes.onboardingVerifyRoute.to }),
  );
  const isOnboardingInvitesPage = Boolean(
    matchRoute({ to: appRoutes.onboardingInvitesRoute.to }),
  );
  const isOnboardingCreateTenantPage = Boolean(
    matchRoute({ to: appRoutes.onboardingCreateTenantRoute.to }),
  );
  const isOnboardingGetStartedPage =
    Boolean(matchRoute({ to: appRoutes.onboardingGetStartedRoute.to })) ||
    Boolean(matchRoute({ to: appRoutes.tenantOnboardingGetStartedRoute.to }));
  const isOnboardingPage =
    isOnboardingVerifyEmailPage ||
    isOnboardingInvitesPage ||
    isOnboardingCreateTenantPage ||
    isOnboardingGetStartedPage;

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to });
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
    const userQueryError = userQuery.error as AxiosError<User> | null;

    // Skip all redirects for organization pages
    if (isOrganizationsPage) {
      return;
    }

    if (userQueryError?.status === 401 || userQueryError?.status === 403) {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (
      userQuery.data &&
      !userQuery.data.emailVerified &&
      !isOnboardingVerifyEmailPage
    ) {
      navigate({ to: appRoutes.onboardingVerifyRoute.to, replace: true });
      return;
    }

    if (
      invitesQuery.data?.length &&
      invitesQuery.data.length > 0 &&
      !isOnboardingInvitesPage
    ) {
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
      return;
    }

    if (listMembershipsQuery.data?.rows?.length === 0 && !isOnboardingPage) {
      navigate({ to: appRoutes.onboardingCreateTenantRoute.to, replace: true });
      return;
    }

    // If user has memberships and we're at the bare root, go to their first tenant
    if (
      pathname === '/' &&
      listMembershipsQuery.data?.rows &&
      listMembershipsQuery.data.rows.length > 0
    ) {
      const memberships = listMembershipsQuery.data.rows;
      const lastTenantId = lastTenant?.metadata.id;

      const lastTenantInMemberships = lastTenantId
        ? memberships.find((m) => m.tenant?.metadata.id === lastTenantId)
            ?.tenant
        : undefined;

      // If the cached tenant isn't in the current user's memberships (e.g. user switched),
      // clear it so we don't keep trying to use a stale tenant.
      if (lastTenantId && !lastTenantInMemberships) {
        setLastTenant(undefined);
      }

      const targetTenant = lastTenantInMemberships ?? memberships[0].tenant;

      if (targetTenant) {
        navigate({
          to: appRoutes.tenantRunsRoute.to,
          params: { tenant: targetTenant.metadata.id },
          replace: true,
        });
      }
    }
  }, [
    tenant?.metadata.id,
    userQuery.data,
    invitesQuery.data,
    listMembershipsQuery.data,
    tenant?.version,
    userQuery.error,
    navigate,
    lastTenant,
    pathname,
    isOrganizationsPage,
    isOnboardingVerifyEmailPage,
    isOnboardingInvitesPage,
    isOnboardingPage,
    setLastTenant,
  ]);

  useEffect(() => {
    if (userQuery.error && !isAuthPage) {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    }
  }, [isAuthPage, navigate, userQuery.error]);

  if (
    userQuery.isLoading ||
    invitesQuery.isLoading ||
    listMembershipsQuery.isLoading
  ) {
    return <Loading />;
  }

  if (userQuery.error && !isAuthPage) {
    return null;
  }

  if (!userQuery.data) {
    return <Loading />;
  }

  // Allow organization pages even without tenant memberships
  if (!isOrganizationsPage && !listMembershipsQuery.data?.rows) {
    return <Loading />;
  }

  return (
    <PostHogProvider user={userQuery.data}>
      <SupportChat user={userQuery.data}>
        <div className="flex h-full w-full flex-1 flex-row">
          <MainNav
            user={userQuery.data}
            tenantMemberships={listMembershipsQuery.data?.rows || []}
          />
          <div className="flex-grow overflow-y-auto overflow-x-hidden pt-16">
            <OutletWithContext context={ctx} />
          </div>
        </div>
      </SupportChat>
    </PostHogProvider>
  );
}
