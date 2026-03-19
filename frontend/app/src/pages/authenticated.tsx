import { AppLayout } from '@/components/layout/app-layout';
import SupportChat from '@/components/support-chat';
import TopNav from '@/components/v1/nav/top-nav.tsx';
import { useCurrentUser } from '@/hooks/use-current-user.ts';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries, User } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useContextFromParent } from '@/lib/outlet';
import { OutletWithContext } from '@/lib/router-helpers';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { PostHogProvider } from '@/providers/posthog';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  useLocation,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { lazy, Suspense, useEffect } from 'react';

const DevtoolsFooter = import.meta.env.DEV
  ? lazy(() => import('../devtools.tsx'))
  : null;

function AuthenticatedInner() {
  const { tenant } = useTenantDetails();
  const {
    currentUser,
    error: userError,
    isLoading: isUserLoading,
  } = useCurrentUser();
  const userApi = useUserApi();

  const { data: cloudMetadata } = useQuery({
    queryKey: ['metadata'],
    queryFn: async () => {
      const res = await cloudApi.metadataGet();
      return res.data;
    },
  });

  const navigate = useNavigate();
  const location = useLocation();
  const matchRoute = useMatchRoute();
  const isAuthPage =
    Boolean(matchRoute({ to: appRoutes.authLoginRoute.to })) ||
    Boolean(matchRoute({ to: appRoutes.authRegisterRoute.to }));
  const isTenantPage = Boolean(
    matchRoute({ to: appRoutes.tenantRoute.to, fuzzy: true }),
  );
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
  const isOnboardingPage =
    isOnboardingVerifyEmailPage ||
    isOnboardingInvitesPage ||
    isOnboardingCreateTenantPage;

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await userApi.userUpdateLogout();
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

  const { pendingInvitesQuery, isLoading: isPendingInvitesLoading } =
    usePendingInvites();

  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
    retry: false,
  });

  const ctx = useContextFromParent({
    user: currentUser,
    memberships: listMembershipsQuery.data?.rows,
  });

  useEffect(() => {
    console.log('[Authenticated] render state', {
      path: window.location.pathname,
      isUserLoading,
      hasCurrentUser: Boolean(currentUser),
      userErrorStatus: (userError as AxiosError | null | undefined)?.status,
      membershipsCount: listMembershipsQuery.data?.rows?.length,
      pendingInvites: pendingInvitesQuery.data,
      isPendingInvitesLoading,
      isAuthPage,
      isTenantPage,
      isOrganizationsPage,
      isOnboardingPage,
      tenantId: tenant?.metadata.id,
    });
  }, [
    isUserLoading,
    currentUser,
    userError,
    listMembershipsQuery.data?.rows?.length,
    pendingInvitesQuery.data,
    isPendingInvitesLoading,
    isAuthPage,
    isTenantPage,
    isOrganizationsPage,
    isOnboardingPage,
    tenant?.metadata.id,
  ]);

  useEffect(() => {
    const userQueryError = userError as AxiosError<User> | null | undefined;
    const isRootPath = location.pathname === '/';

    // Skip all redirects for organization pages
    if (isOrganizationsPage) {
      console.log('[Authenticated] skip redirects on organizations page');
      return;
    }

    // If we definitively have no user, always go to login.
    if (!isUserLoading && !currentUser && !isAuthPage) {
      console.log('[Authenticated] redirect -> login (no current user)');
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (userQueryError?.status === 401 || userQueryError?.status === 403) {
      console.log('[Authenticated] redirect -> login (401/403 user query)');
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (
      currentUser &&
      !currentUser.emailVerified &&
      !isOnboardingVerifyEmailPage
    ) {
      console.log(
        '[Authenticated] redirect -> onboarding verify (email not verified)',
      );
      navigate({ to: appRoutes.onboardingVerifyRoute.to, replace: true });
      return;
    }

    if (
      pendingInvitesQuery.data &&
      pendingInvitesQuery.data > 0 &&
      !isOnboardingInvitesPage
    ) {
      console.log(
        '[Authenticated] redirect -> onboarding invites (pending invites)',
      );
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
      return;
    }

    if (
      !isPendingInvitesLoading &&
      listMembershipsQuery.data?.rows?.length === 0 &&
      !isOnboardingPage
    ) {
      console.log(
        '[Authenticated] redirect -> onboarding create tenant (no memberships)',
      );
      navigate({ to: appRoutes.onboardingCreateTenantRoute.to, replace: true });
      return;
    }

    if (isRootPath && !isUserLoading && currentUser) {
      const firstMembershipTenantId =
        listMembershipsQuery.data?.rows?.[0]?.tenant?.metadata.id;

      if (firstMembershipTenantId) {
        console.log('[Authenticated] redirect -> tenant runs (root path)', {
          tenant: firstMembershipTenantId,
        });
        navigate({
          to: appRoutes.tenantRunsRoute.to,
          params: { tenant: firstMembershipTenantId },
          replace: true,
        });
        return;
      }
    }

    console.log('[Authenticated] no redirect');
  }, [
    location.pathname,
    tenant?.metadata.id,
    currentUser,
    pendingInvitesQuery.data,
    isPendingInvitesLoading,
    listMembershipsQuery.data,
    tenant?.version,
    userError,
    isUserLoading,
    navigate,
    isOrganizationsPage,
    isOnboardingVerifyEmailPage,
    isOnboardingInvitesPage,
    isOnboardingPage,
    isAuthPage,
  ]);

  useEffect(() => {
    if (userError && !isAuthPage) {
      console.log('[Authenticated] fallback redirect -> login (userError)');
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    }
  }, [isAuthPage, navigate, userError]);

  return (
    <PostHogProvider user={currentUser}>
      <SupportChat user={currentUser}>
        <AppLayout
          header={
            <TopNav
              user={currentUser}
              tenantMemberships={listMembershipsQuery.data?.rows || []}
            />
          }
          footer={
            isTenantPage && DevtoolsFooter ? (
              <Suspense fallback={null}>
                <DevtoolsFooter />
              </Suspense>
            ) : undefined
          }
          // Tenant routes (v1 shell) own their internal scrolling; everything else scrolls here.
          contentScroll={!isTenantPage}
        >
          <OutletWithContext context={ctx} />
        </AppLayout>
      </SupportChat>
    </PostHogProvider>
  );
}

export default function Authenticated() {
  return <AuthenticatedInner />;
}
