import { getCloudMetadataQuery } from '../hooks/use-cloud.ts';
import { NewTenantSaverForm } from '@/components/forms/new-tenant-saver-form';
import { AppLayout } from '@/components/layout/app-layout';
import { AuthDisabledBanner } from '@/components/layout/auth-disabled-banner';
import { AddOrgMemberToTenantModal } from '@/components/modals/add-org-member-to-tenant-modal';
import { CreateTenantInviteModal } from '@/components/modals/create-tenant-invite-modal';
import { InviteModal } from '@/components/modals/invite-modal';
import { OrganizationInviteMemberModal } from '@/components/modals/organization-invite-member-modal';
import { WelcomeModal } from '@/components/modals/welcome-modal';
import {
  readWelcomeTrigger,
  WELCOME_KEY,
  WELCOME_TRIGGER,
} from '@/components/modals/welcome-modal-state';
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
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentUser } from '@/hooks/use-current-user.ts';
import {
  pendingInvitesQuery,
  usePendingInvites,
} from '@/hooks/use-pending-invites.ts';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { User, queries } from '@/lib/api';
import {
  CONTROL_PLANE_TENANT_STORAGE_KEY,
  fetchControlPlaneStatus,
} from '@/lib/api/api';
import { SubscriptionPlanCode } from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useUserApi } from '@/lib/api/user-wrapper';
import { lastTenantAtom } from '@/lib/atoms';
import { globalEmitter } from '@/lib/global-emitter';
import { useContextFromParent } from '@/lib/outlet';
import { REDIRECT_TARGET_KEY } from '@/lib/redirect';
import { OutletWithContext } from '@/lib/router-helpers';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useInactivityDetection } from '@/pages/auth/hooks/use-inactivity-detection';
import { PostHogProvider } from '@/providers/posthog';
import { useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
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
  const { isControlPlaneEnabled } = await fetchControlPlaneStatus();

  const { isCloudEnabled, ...meta } = isControlPlaneEnabled
    ? { isCloudEnabled: false as const }
    : await queryClient.fetchQuery(getCloudMetadataQuery);

  await queryClient.fetchQuery(
    pendingInvitesQuery(isCloudEnabled, isControlPlaneEnabled),
  );
  return {
    isCloudEnabled,
    isControlPlaneEnabled,
    inactivityLogoutMs:
      'inactivityLogoutMs' in meta ? (meta.inactivityLogoutMs ?? -1) : -1,
  };
}

function AuthenticatedInner() {
  const { tenant, organizationId } = useTenantDetails();
  const { meta } = useApiMeta();
  const { capture } = useAnalytics();
  const {
    currentUser,
    error: userError,
    isLoading: isUserLoading,
  } = useCurrentUser();
  const [lastTenant, setLastTenant] = useAtom(lastTenantAtom);
  const [authBannerDismissed, setAuthBannerDismissed] = useState(() => {
    try {
      return localStorage.getItem('auth-disabled-banner-dismissed') === 'true';
    } catch {
      return false;
    }
  });
  const [newTenantModalOpen, setNewTenantModalOpen] = useState(false);
  const [defaultOrganizationId, setDefaultOrganizationId] = useState<
    string | undefined
  >();
  const [newTenantAllTags, setNewTenantAllTags] = useState<string[]>([]);
  const [inviteModalTenantId, setInviteModalTenantId] = useState<
    string | undefined
  >();
  const [orgInviteModal, setOrgInviteModal] = useState<
    { organizationId: string; organizationName: string } | undefined
  >();
  const [addOrgMemberToTenantModal, setAddOrgMemberToTenantModal] = useState<
    { tenantId: string; organizationId: string } | undefined
  >();
  const [showWelcome, setShowWelcome] = useState(false);
  const [inviteModalOpen, setInviteModalOpen] = useState(false);

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
  const isTenantsPage = Boolean(matchRoute({ to: appRoutes.tenantsRoute.to }));
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
  const isOnboardingNoTenantsPage = Boolean(
    matchRoute({ to: appRoutes.onboardingNoTenantsRoute.to }),
  );
  const isOnboardingPage =
    isOnboardingVerifyEmailPage ||
    isOnboardingInvitesPage ||
    isOnboardingCreateTenantPage ||
    isOnboardingCreateOrganizationPage ||
    isOnboardingNoTenantsPage;

  const { userUpdateLogoutMutation } = useUserApi();
  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      // always clear on logout attempt, even if the request fails
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  const { pendingInvitesQuery } = usePendingInvites({
    isCloudEnabled: loaderData.isCloudEnabled,
    isControlPlaneEnabled: loaderData.isControlPlaneEnabled,
  });

  const {
    isCloudEnabled,
    isLoaded: isUserUniverseLoaded,
    isFetching: isUserUniverseFetching,
    organizations,
    tenantMemberships,
  } = useUserUniverse();

  const { controlPlaneMeta, isControlPlaneEnabled } = useControlPlane();
  const orgApi = useOrganizationApi();
  const orgIdForTenant = organizations?.find((o) =>
    o.tenants?.some((t) => t.id === tenant?.metadata?.id),
  )?.metadata?.id;
  const orgQuery = useQuery({
    ...orgApi.organizationGetQuery(orgIdForTenant!),
    enabled: !!orgIdForTenant && isControlPlaneEnabled,
  });
  const inactivityTimeoutMs = isControlPlaneEnabled
    ? ((orgQuery.data as { inactivity_timeout?: number } | undefined)
        ?.inactivity_timeout ?? -1)
    : loaderData.inactivityLogoutMs;
  const welcomeBillingState = useQuery({
    ...queries.controlPlane.billing(organizationId || ''),
    enabled:
      isCloudEnabled &&
      isControlPlaneEnabled &&
      !!controlPlaneMeta?.canBill &&
      !!organizationId,
    retry: false,
  });

  useInactivityDetection({
    timeoutMs: inactivityTimeoutMs,
    onInactive: () => {
      logoutMutation.mutate();
    },
  });
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

    // Skip all redirects for organization/tenants pages
    if (isOrganizationsPage || isTenantsPage) {
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

    const okayToMakeOnboardingRedirectDecisions =
      pendingInvitesQuery.isSuccess &&
      !isOnboardingPage &&
      isUserUniverseLoaded &&
      !isUserUniverseFetching;

    const shouldHaveAnOrganizationButDoesnt =
      isCloudEnabled && isUserUniverseLoaded && organizations.length === 0;

    // Redirect to invites page only for users with no memberships yet (new users).
    // Existing users with memberships see the InviteModal overlay instead.
    const mustRedirectToInvitesPage =
      okayToMakeOnboardingRedirectDecisions &&
      (tenantMemberships?.length ?? 0) === 0 &&
      pendingInvites &&
      pendingInvites.inviteCount > 0;

    if (!isOnboardingInvitesPage && mustRedirectToInvitesPage) {
      navigate({ to: appRoutes.onboardingInvitesRoute.to, replace: true });
      return;
    } else if (
      okayToMakeOnboardingRedirectDecisions &&
      shouldHaveAnOrganizationButDoesnt
    ) {
      storeRedirectPath();
      navigate({
        to: appRoutes.onboardingCreateOrganizationRoute.to,
        replace: true,
      });
      return;
    } else if (
      okayToMakeOnboardingRedirectDecisions &&
      isControlPlaneEnabled &&
      organizations &&
      organizations.length > 0 &&
      tenantMemberships.length === 0
    ) {
      storeRedirectPath();
      navigate({
        to: appRoutes.onboardingNoTenantsRoute.to,
        replace: true,
      });
      return;
    } else if (
      okayToMakeOnboardingRedirectDecisions &&
      tenantMemberships.length === 0
    ) {
      storeRedirectPath();
      navigate({
        to: appRoutes.onboardingCreateTenantRoute.to,
        replace: true,
      });
      return;
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
        if (loaderData.isControlPlaneEnabled) {
          localStorage.removeItem(CONTROL_PLANE_TENANT_STORAGE_KEY);
        }
      }

      const targetTenant =
        lastTenantInMemberships ?? tenantMemberships[0].tenant;

      if (targetTenant) {
        if (loaderData.isControlPlaneEnabled) {
          localStorage.setItem(
            CONTROL_PLANE_TENANT_STORAGE_KEY,
            JSON.stringify(targetTenant),
          );
        }

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
    isTenantsPage,
    isOnboardingVerifyEmailPage,
    isOnboardingInvitesPage,
    isOnboardingPage,
    isAuthPage,
    setLastTenant,
    isCloudEnabled,
    isUserUniverseLoaded,
    isUserUniverseFetching,
    organizations,
    isOnboardingCreateOrganizationPage,
    isOnboardingCreateTenantPage,
    isControlPlaneEnabled,
    loaderData.isControlPlaneEnabled,
  ]);

  useEffect(
    () =>
      globalEmitter.on(
        'create-new-tenant',
        ({ defaultOrganizationId, allTenantTags }) => {
          setDefaultOrganizationId(defaultOrganizationId);
          setNewTenantAllTags(allTenantTags ?? []);
          setNewTenantModalOpen(true);
        },
      ),
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

  useEffect(
    () =>
      globalEmitter.on('open-invite-modal', () => {
        setInviteModalOpen(true);
      }),
    [],
  );

  useEffect(
    () =>
      globalEmitter.on(
        'add-org-member-to-tenant',
        ({ tenantId, organizationId }) => {
          setAddOrgMemberToTenantModal({ tenantId, organizationId });
        },
      ),
    [],
  );

  useEffect(() => {
    const welcomeTrigger = readWelcomeTrigger(
      localStorage.getItem(WELCOME_KEY),
    );
    if (!welcomeTrigger) {
      return;
    }

    if (!tenant?.metadata.id) {
      return;
    }

    if (!isUserUniverseLoaded) {
      return;
    }

    if (!isCloudEnabled) {
      localStorage.removeItem(WELCOME_KEY);
      return;
    }

    if (!organizationId) {
      return;
    }

    if (!controlPlaneMeta?.canBill) {
      return;
    }

    if (welcomeTrigger === WELCOME_TRIGGER.OrganizationCreated) {
      localStorage.removeItem(WELCOME_KEY);
      setShowWelcome(true);
      capture('welcome_modal_shown', {
        tenant_id: tenant?.metadata.id,
        source: welcomeTrigger,
      });
      return;
    }

    if (welcomeBillingState.isPending) {
      return;
    }

    const billingStateError =
      welcomeBillingState.error as AxiosError<unknown> | null;
    const billingStateNotFound =
      billingStateError?.status === 404 ||
      billingStateError?.response?.status === 404;

    if (welcomeBillingState.isError && !billingStateNotFound) {
      return;
    }

    const currentSubscription = billingStateNotFound
      ? undefined
      : welcomeBillingState.data?.currentSubscription;
    const canShowWelcomeForSubscription =
      !currentSubscription ||
      currentSubscription.plan === SubscriptionPlanCode.Free;

    if (!canShowWelcomeForSubscription) {
      localStorage.removeItem(WELCOME_KEY);
      return;
    }

    localStorage.removeItem(WELCOME_KEY);
    setShowWelcome(true);
    capture('welcome_modal_shown', {
      tenant_id: tenant?.metadata.id,
      source: welcomeTrigger,
    });
  }, [
    tenant?.metadata.id,
    organizationId,
    capture,
    isCloudEnabled,
    isUserUniverseLoaded,
    controlPlaneMeta?.canBill,
    welcomeBillingState.data?.currentSubscription,
    welcomeBillingState.error,
    welcomeBillingState.isError,
    welcomeBillingState.isPending,
  ]);

  // Auto-open invite modal for users who already have memberships when new invites appear.
  // New users with no memberships are redirected to /onboarding/invites instead.
  // Requires settled data (!isFetching): while a post-accept refetch is in
  // flight the cached inviteCount is stale, and opening on it would resurface
  // an already-processed invite.
  useEffect(() => {
    if (
      !isOnboardingInvitesPage &&
      pendingInvitesQuery.isSuccess &&
      !pendingInvitesQuery.isFetching &&
      (pendingInvitesQuery.data?.inviteCount ?? 0) > 0 &&
      (tenantMemberships?.length ?? 0) > 0
    ) {
      setInviteModalOpen(true);
    }
  }, [
    pendingInvitesQuery.isSuccess,
    pendingInvitesQuery.isFetching,
    pendingInvitesQuery.data?.inviteCount,
    isOnboardingInvitesPage,
    tenantMemberships?.length,
  ]);

  if (!currentUser) {
    return <Loading />;
  }

  return (
    <PostHogProvider user={currentUser}>
      <SupportChat user={currentUser}>
        <AppLayout
          banner={
            meta &&
            'authDisabled' in meta &&
            meta.authDisabled &&
            !authBannerDismissed ? (
              <AuthDisabledBanner
                onDismiss={() => {
                  try {
                    localStorage.setItem(
                      'auth-disabled-banner-dismissed',
                      'true',
                    );
                  } catch {
                    /* empty */
                  }
                  setAuthBannerDismissed(true);
                }}
              />
            ) : undefined
          }
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
                allTenantTags={newTenantAllTags}
                afterSave={(result) => {
                  setDefaultOrganizationId(undefined);
                  setNewTenantAllTags([]);
                  setNewTenantModalOpen(false);
                  const tenantId =
                    result.type === 'cloud'
                      ? result.tenant.id
                      : result.tenant.metadata.id;

                  if (result.type === 'cloud') {
                    void queryClient.prefetchQuery(
                      queries.controlPlane.subscriptionPlans(),
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
        {addOrgMemberToTenantModal && (
          <AddOrgMemberToTenantModal
            organizationId={addOrgMemberToTenantModal.organizationId}
            tenantId={addOrgMemberToTenantModal.tenantId}
            onClose={() => setAddOrgMemberToTenantModal(undefined)}
          />
        )}
        <WelcomeModal
          tenantId={tenant?.metadata.id}
          organizationId={organizationId}
          open={showWelcome}
          onClose={() => setShowWelcome(false)}
        />
        <InviteModal
          isOpen={inviteModalOpen}
          onClose={() => setInviteModalOpen(false)}
        />
      </SupportChat>
    </PostHogProvider>
  );
}

export default function Authenticated() {
  return <AuthenticatedInner />;
}
