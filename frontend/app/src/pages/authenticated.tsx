import { AppLayout } from '@/components/layout/app-layout';
import SupportChat from '@/components/support-chat';
import TopNav from '@/components/v1/nav/top-nav.tsx';
import { useCurrentUser } from '@/hooks/use-current-user.ts';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { queries, User } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { lastTenantAtom } from '@/lib/atoms';
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
import { useAtom } from 'jotai';
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
    const userQueryError = userError as AxiosError<User> | null | undefined;

    // Skip all redirects for organization pages
    if (isOrganizationsPage) {
      return;
    }

    // If we definitively have no user, always go to login.
    if (!isUserLoading && !currentUser && !isAuthPage) {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (userQueryError?.status === 401 || userQueryError?.status === 403) {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (
      currentUser &&
      !currentUser.emailVerified &&
      !isOnboardingVerifyEmailPage
    ) {
      navigate({ to: appRoutes.onboardingVerifyRoute.to, replace: true });
      return;
    }

    if (
      pendingInvitesQuery.data &&
      pendingInvitesQuery.data > 0 &&
      !isOnboardingInvitesPage
    ) {
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
      return;
    }

    if (
      !isPendingInvitesLoading &&
      listMembershipsQuery.data?.rows?.length === 0 &&
      !isOnboardingPage
    ) {
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
        // Check if tenant has workflows to decide where to redirect
        api
          .workflowList(targetTenant.metadata.id, { limit: 1 })
          .then((response) => {
            const hasWorkflows =
              response.data.rows && response.data.rows.length > 0;

            navigate({
              to: hasWorkflows
                ? appRoutes.tenantRunsRoute.to
                : appRoutes.tenantOverviewRoute.to,
              params: { tenant: targetTenant.metadata.id },
              replace: true,
            });
          })
          .catch(() => {
            // On error, default to runs page
            navigate({
              to: appRoutes.tenantRunsRoute.to,
              params: { tenant: targetTenant.metadata.id },
              replace: true,
            });
          });
      }
    }
  }, [
    tenant?.metadata.id,
    currentUser,
    pendingInvitesQuery.data,
    isPendingInvitesLoading,
    listMembershipsQuery.data,
    tenant?.version,
    userError,
    isUserLoading,
    navigate,
    lastTenant,
    pathname,
    isOrganizationsPage,
    isOnboardingVerifyEmailPage,
    isOnboardingInvitesPage,
    isOnboardingPage,
    isAuthPage,
    setLastTenant,
  ]);

  useEffect(() => {
    if (userError && !isAuthPage) {
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
