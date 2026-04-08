import { getCloudMetadataQuery } from '../hooks/use-cloud.ts';
import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { AppLayout } from '@/components/layout/app-layout';
import { CreateTenantInviteModal } from '@/components/modals/create-tenant-invite-modal';
import { OrganizationInviteMemberModal } from '@/components/modals/organization-invite-member-modal';
import { WelcomeModal } from '@/components/modals/welcome-modal';
import SupportChat from '@/components/support-chat';
import TopNav from '@/components/v1/nav/top-nav.tsx';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useAnalytics } from '@/hooks/use-analytics';
import { useCurrentUser } from '@/hooks/use-current-user.ts';
import {
  pendingInvitesQuery,
  usePendingInvites,
} from '@/hooks/use-pending-invites.ts';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { User, queries } from '@/lib/api';
import { fetchControlPlaneStatus } from '@/lib/api/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { lastTenantAtom } from '@/lib/atoms';
import { globalEmitter } from '@/lib/global-emitter';
import { useContextFromParent } from '@/lib/outlet';
import { REDIRECT_TARGET_KEY } from '@/lib/redirect';
import { OutletWithContext } from '@/lib/router-helpers';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { PostHogProvider } from '@/providers/posthog';
import { useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import {
  useLoaderData,
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

export async function loader(_args: { request: Request }) {
  const [{ isCloudEnabled, ...meta }, { isControlPlaneEnabled }] =
    await Promise.all([
      queryClient.fetchQuery(getCloudMetadataQuery),
      fetchControlPlaneStatus(),
    ]);

  await queryClient.fetchQuery(
    pendingInvitesQuery(isCloudEnabled, isControlPlaneEnabled),
  );
  return {
    inactivityLogoutMs:
      'inactivityLogoutMs' in meta ? (meta.inactivityLogoutMs ?? -1) : -1,
  };
}

function AuthenticatedInner() {
  const { tenant } = useTenantDetails();
  const { capture } = useAnalytics();
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
  const [inviteModalTenantId, setInviteModalTenantId] = useState<
    string | undefined
  >();
  const [orgInviteModal, setOrgInviteModal] = useState<
    { organizationId: string; organizationName: string } | undefined
  >();
  const [showWelcome, setShowWelcome] = useState(false);

  const loaderData = useLoaderData({ from: '/' });

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

  const { userUpdateLogoutMutation } = useUserApi();
  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSuccess: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  useInactivityDetection({
    timeoutMs: loaderData.inactivityLogoutMs,
    onInactive: () => {
      logoutMutation.mutate();
    },
  });

  const { pendingInvitesQuery } = usePendingInvites();

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

    const storeRedirectPath = () => {
      if (
        pathname !== '/' &&
        !pathname.startsWith('/onboarding/') &&
        !pathname.startsWith('/auth/')
      ) {
        sessionStorage.setItem(REDIRECT_TARGET_KEY, pathname);
      }
    };

    // Skip all redirects for organization pages
    if (isOrganizationsPage) {
      return;
    }

    // If we definitively have no user, always go to login.
    if (!isUserLoading && !currentUser && !isAuthPage) {
      storeRedirectPath();
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
      return;
    }

    if (userQueryError?.status === 401 || userQueryError?.status === 403) {
      storeRedirectPath();
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

    const pendingInvites = pendingInvitesQuery.isSuccess
      ? pendingInvitesQuery.data
      : null;

    if (
      pendingInvites &&
      pendingInvites.inviteCount > 0 &&
      !isOnboardingInvitesPage
    ) {
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
      return;
    }

    const okayToMakeOnboardingRedirectDecisions =
      pendingInvitesQuery.isSuccess &&
      !isOnboardingPage &&
      isUserUniverseLoaded;

    if (okayToMakeOnboardingRedirectDecisions) {
      const shouldHaveAnOrganizationButDoesnt =
        isCloudEnabled && organizations.length === 0;

      if (shouldHaveAnOrganizationButDoesnt) {
        storeRedirectPath();
        navigate({
          to: appRoutes.onboardingCreateOrganizationRoute.to,
          replace: true,
        });
        return;
      }

      if (tenantMemberships.length === 0) {
        storeRedirectPath();
        navigate({
          to: appRoutes.onboardingCreateTenantRoute.to,
          replace: true,
        });
        return;
      }
    }

    // If user has memberships and we're at the bare root, go to their first tenant
    if (pathname === '/' && tenantMemberships && tenantMemberships.length > 0) {
      const savedRedirect = sessionStorage.getItem(REDIRECT_TARGET_KEY);
      if (savedRedirect) {
        sessionStorage.removeItem(REDIRECT_TARGET_KEY);
        navigate({ to: savedRedirect, replace: true } as never);
        return;
      }

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
    pendingInvitesQuery,
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
    isOnboardingCreateOrganizationPage,
    isOnboardingCreateTenantPage,
  ]);

  useEffect(
    () =>
      globalEmitter.on('create-new-tenant', ({ defaultOrganizationId }) => {
        setDefaultOrganizationId(defaultOrganizationId);
        setNewTenantModalOpen(true);
      }),
    [],
  );

  useEffect(
    () =>
      globalEmitter.on('create-tenant-invite', ({ tenantId }) => {
        setInviteModalTenantId(tenantId);
      }),
    [],
  );

  useEffect(
    () =>
      globalEmitter.on(
        'create-organization-invite',
        ({ organizationId, organizationName }) => {
          setOrgInviteModal({ organizationId, organizationName });
        },
      ),
    [],
  );

  useEffect(() => {
    const key = 'hatchet:show-welcome';
    if (localStorage.getItem(key)) {
      localStorage.removeItem(key);
      setShowWelcome(true);
      capture('welcome_modal_shown', {
        tenant_id: tenant?.metadata.id,
      });
    }
  }, [tenant?.metadata.id, capture]);

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

                  if (result.type === 'cloud') {
                    void queryClient.prefetchQuery(
                      queries.cloud.subscriptionPlans(),
                    );
                  }

                  navigate({
                    to: appRoutes.tenantOverviewRoute.to,
                    params: { tenant: tenantId },
                  });
                }}
              />
            </div>
          </DialogContent>
        </Dialog>
        {inviteModalTenantId && (
          <CreateTenantInviteModal
            tenantId={inviteModalTenantId}
            onClose={() => setInviteModalTenantId(undefined)}
            onCreated={(invite) => {
              globalEmitter.emit('tenant-invite-created', {
                tenantId: inviteModalTenantId,
                invite,
              });
            }}
          />
        )}
        {orgInviteModal && (
          <OrganizationInviteMemberModal
            organizationId={orgInviteModal.organizationId}
            organizationName={orgInviteModal.organizationName}
            onClose={() => setOrgInviteModal(undefined)}
            onCreated={(invite) => {
              globalEmitter.emit('organization-invite-created', {
                organizationId: orgInviteModal.organizationId,
                invite,
              });
            }}
          />
        )}
        <WelcomeModal
          tenantId={tenant?.metadata.id}
          open={showWelcome}
          onClose={() => setShowWelcome(false)}
        />
      </SupportChat>
    </PostHogProvider>
  );
}

export default function Authenticated() {
  return <AuthenticatedInner />;
}
