import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { AppLayout } from '@/components/layout/app-layout';
import SupportChat from '@/components/support-chat';
import TopNav from '@/components/v1/nav/top-nav.tsx';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useCurrentUser } from '@/hooks/use-current-user.ts';
import { usePendingInvites } from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { User } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { lastTenantAtom } from '@/lib/atoms';
import { globalEmitter } from '@/lib/global-emitter';
import { useContextFromParent } from '@/lib/outlet';
import { OutletWithContext } from '@/lib/router-helpers';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { PostHogProvider } from '@/providers/posthog';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  useLocation,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useAtom } from 'jotai';
import { lazy, Suspense, useEffect, useState } from 'react';

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
  const [newTenantModalOpen, setNewTenantModalOpen] = useState(false);
  const [defaultOrganizationId, setDefaultOrganizationId] = useState<
    string | undefined
  >();

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
  const isOnboardingCreateOrganizationPage = Boolean(
    matchRoute({ to: appRoutes.onboardingCreateOrganizationRoute.to }),
  );
  const isOnboardingPage =
    isOnboardingVerifyEmailPage ||
    isOnboardingInvitesPage ||
    isOnboardingCreateTenantPage ||
    isOnboardingCreateOrganizationPage;

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

  const {
    isCloudEnabled,
    isLoaded: isUserUniverseLoaded,
    organizations,
    tenantMemberships,
  } = useUserUniverse();

  const ctx = useContextFromParent({
    user: currentUser,
    memberships: tenantMemberships,
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

    const okayToMakeOnboardingRedirectDecisions =
      !isPendingInvitesLoading && !isOnboardingPage && isUserUniverseLoaded;

    if (okayToMakeOnboardingRedirectDecisions) {
      const shouldHaveAnOrganizationButDoesnt =
        isCloudEnabled && organizations.length === 0;

      if (shouldHaveAnOrganizationButDoesnt) {
        navigate({
          to: appRoutes.onboardingCreateOrganizationRoute.to,
          replace: true,
        });
        return;
      }

      if (tenantMemberships.length === 0) {
        navigate({
          to: appRoutes.onboardingCreateTenantRoute.to,
          replace: true,
        });
        return;
      }
    }

    // If user has memberships and we're at the bare root, go to their first tenant
    if (pathname === '/' && tenantMemberships && tenantMemberships.length > 0) {
      const lastTenantId = lastTenant?.metadata.id;

      const lastTenantInMemberships = lastTenantId
        ? tenantMemberships.find((m) => m.tenant?.metadata.id === lastTenantId)
            ?.tenant
        : undefined;

      // If the cached tenant isn't in the current user's memberships (e.g. user switched),
      // clear it so we don't keep trying to use a stale tenant.
      if (lastTenantId && !lastTenantInMemberships) {
        setLastTenant(undefined);
      }

      const targetTenant =
        lastTenantInMemberships ?? tenantMemberships[0].tenant;

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
    tenantMemberships,
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
    isCloudEnabled,
    isUserUniverseLoaded,
    organizations,
  ]);

  useEffect(() => {
    if (userError && !isAuthPage) {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    }
  }, [isAuthPage, navigate, userError]);

  useEffect(
    () =>
      globalEmitter.on('new-tenant', ({ defaultOrganizationId }) => {
        setDefaultOrganizationId(defaultOrganizationId);
        setNewTenantModalOpen(true);
      }),
    [],
  );

  if (!currentUser) {
    return <Loading />;
  }

  return (
    <PostHogProvider user={currentUser}>
      <SupportChat user={currentUser}>
        <AppLayout
          header={
            <TopNav
              user={currentUser}
              tenantMemberships={tenantMemberships || []}
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

        <Dialog open={newTenantModalOpen} onOpenChange={setNewTenantModalOpen}>
          <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
            <DialogHeader>
              <DialogTitle>Create New Tenant</DialogTitle>
            </DialogHeader>
            <div className="flex justify-center">
              <NewTenantSaverForm
                defaultOrganizationId={defaultOrganizationId}
                afterSave={(result) => {
                  setDefaultOrganizationId(undefined);
                  setNewTenantModalOpen(false);
                  const tenantId =
                    result.type === 'cloud'
                      ? result.tenant.id
                      : result.tenant.metadata.id;
                  navigate({
                    to: appRoutes.tenantOverviewRoute.to,
                    params: { tenant: tenantId },
                  });
                }}
              />
            </div>
          </DialogContent>
        </Dialog>
      </SupportChat>
    </PostHogProvider>
  );
}

export default function Authenticated() {
  return <AuthenticatedInner />;
}
