import { getCloudMetadataQuery } from './pages/auth/hooks/use-cloud';
import { NotFound } from './pages/error/components/not-found';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.tsx';
import { userUniverseQuery } from './providers/user-universe';
import api from '@/lib/api';
import queryClient from '@/query-client';
import {
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  redirect,
} from '@tanstack/react-router';
import { Outlet } from '@tanstack/react-router';
import { FC } from 'react';
import { validate } from 'uuid';

const rootRoute = createRootRoute({
  component: Root,
  errorComponent: (props) => (
    <Root>
      <ErrorBoundary {...props} />
    </Root>
  ),
  notFoundComponent: () => (
    <Root>
      <NotFound />
    </Root>
  ),
});

const authRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: 'auth',
  loader: async () => {
    const mod = await import('./pages/auth/no-auth');
    if (mod.loader) {
      return mod.loader();
    }
    return null;
  },
  component: () => (
    <div className="h-full w-full overflow-y-auto overflow-x-hidden">
      <Outlet />
    </div>
  ),
});

const authLoginRoute = createRoute({
  getParentRoute: () => authRoute,
  path: 'login',
  component: lazyRouteComponent(() => import('./pages/auth/login'), 'default'),
});

const authRegisterRoute = createRoute({
  getParentRoute: () => authRoute,
  path: 'register',
  component: lazyRouteComponent(
    () => import('./pages/auth/register'),
    'default',
  ),
});

const onboardingVerifyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: 'onboarding/verify-email',
  loader: async () => {
    const mod = await import('./pages/onboarding/verify-email');
    if (mod.loader) {
      return mod.loader({
        request: new Request(window.location.href),
        params: {},
      } as never);
    }
    return null;
  },
  component: lazyRouteComponent(
    () => import('./pages/onboarding/verify-email'),
    'default',
  ),
});

const organizationsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'organizations/$organization',
  component: lazyRouteComponent(
    () => import('./pages/organizations/$organization'),
    'default',
  ),
});

const organizationsNewRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'organizations/new',
  component: lazyRouteComponent(
    () => import('./pages/organizations/new'),
    'default',
  ),
});

const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: lazyRouteComponent(
    () => import('./pages/authenticated'),
    'default',
  ),
  notFoundComponent: () => <NotFound />,
});

const onboardingCreateTenantRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'onboarding/create-tenant',
  component: lazyRouteComponent(
    () => import('./pages/onboarding/create-tenant'),
    'default',
  ),
  loader: async () => {
    const { isCloudEnabled } = await queryClient.fetchQuery(
      getCloudMetadataQuery,
    );
    return queryClient.fetchQuery(
      userUniverseQuery({ isCloudEnabled, isCloudLoaded: true }),
    );
  },
});

const onboardingCreateOrganizationRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'onboarding/create-organization',
  component: lazyRouteComponent(
    () => import('./pages/onboarding/create-organization'),
    'default',
  ),
});

const onboardingInvitesRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'onboarding/invites',
  loader: async () => {
    const mod = await import('./pages/onboarding/invites');
    if (mod.loader) {
      return mod.loader({} as never);
    }
    return null;
  },
  component: lazyRouteComponent(
    () => import('./pages/onboarding/invites'),
    'default',
  ),
});

const v1RedirectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: 'v1/*',
  loader: () => {
    throw redirect({ to: appRoutes.authenticatedRoute.to });
  },
});

const tenantRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'tenants/$tenant',
  loader: async ({ params }) => {
    // Ensure the tenant in the URL is one the user actually has access to.
    // If not, throw a 403 so the global error boundary can show a friendly message.
    const { data: memberships } = await api.tenantMembershipsList();

    const hasAccess = Boolean(
      memberships.rows?.some((m) => m.tenant?.metadata.id === params.tenant),
    );

    if (!hasAccess) {
      throw new Response('Forbidden', { status: 403, statusText: 'Forbidden' });
    }

    // Optionally warm the tenant details cache, since most tenant pages expect it.
    // If this fails for any reason, let the error boundary handle it.
    await queryClient.fetchQuery({
      queryKey: ['tenant:get', params.tenant],
      queryFn: async () => (await api.tenantGet(params.tenant)).data,
      retry: false,
    });

    return null;
  },
  component: lazyRouteComponent(() => import('./pages/main/v1'), 'default'),
  notFoundComponent: () => <NotFound />,
});

const tenantIndexRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: '/',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantRunsRoute.to, params });
  },
});

const tenantEventsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'events',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/events'),
    'default',
  ),
});

const tenantFiltersRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'filters',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/filters'),
    'default',
  ),
});

const tenantWebhooksRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'webhooks',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/webhooks'),
    'default',
  ),
});

const tenantRateLimitsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'rate-limits',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/rate-limits'),
    'default',
  ),
});

const tenantScheduledRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'scheduled',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/scheduled-runs'),
    'default',
  ),
});

const tenantCronJobsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'cron-jobs',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/recurring'),
    'default',
  ),
});

const tenantWorkflowsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflows',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workflows'),
    'default',
  ),
});

const tenantWorkflowRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflows/$workflow',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workflows/$workflow'),
    'default',
  ),
});

const tenantOverviewRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'overview',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/overview/index.tsx'),
    'default',
  ),
});

const tenantRunsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'runs',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workflow-runs-v1/index.tsx'),
    'default',
  ),
});

const tenantRunRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'runs/$run',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workflow-runs-v1/$run'),
    'default',
  ),
});

const tenantTaskRunsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'task-runs/$run',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantRunRoute.to, params });
  },
});

const tenantWorkersRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workers',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workers'),
    'default',
  ),
});

const tenantWorkersAllRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workers/all',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantWorkersRoute.to, params });
  },
});

const tenantWorkerRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workers/$worker',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workers/$worker'),
    'default',
  ),
});

const tenantManagedWorkersRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'managed-workers',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/managed-workers/index.tsx'),
    'default',
  ),
});

const tenantManagedWorkersTemplateRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'managed-workers/demo-template',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/managed-workers/demo-template/index.tsx'),
    'default',
  ),
});

const tenantManagedWorkersCreateRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'managed-workers/create',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/managed-workers/create/index.tsx'),
    'default',
  ),
});

const tenantManagedWorkerRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'managed-workers/$managedWorker',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/managed-workers/$managed-worker/index.tsx'),
    'default',
  ),
});

const tenantSettingsIndexRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings',
  loader: ({ params }) => {
    throw redirect({
      to: appRoutes.tenantSettingsOverviewRoute.to,
      params,
    });
  },
});

const tenantSettingsOverviewRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/overview',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/overview'),
    'default',
  ),
});

const tenantSettingsApiTokensRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/api-tokens',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/api-tokens'),
    'default',
  ),
});

const tenantSettingsGithubRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/github',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/github'),
    'default',
  ),
});

const tenantSettingsMembersRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/members',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/members'),
    'default',
  ),
});

const tenantSettingsAlertingRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/alerting',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/alerting'),
    'default',
  ),
});

const tenantSettingsBillingRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/billing-and-limits',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/resource-limits'),
    'default',
  ),
});

const tenantSettingsIngestorsRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tenant-settings/ingestors',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/tenant-settings/ingestors'),
    'default',
  ),
});

const tenantWorkflowRunsRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflow-runs',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantRunsRoute.to, params });
  },
});

const tenantWorkflowRunRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflow-runs/$run',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantRunsRoute.to, params });
  },
});

const tenantTasksRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tasks',
  loader: ({ params }) => {
    throw redirect({ to: appRoutes.tenantWorkflowsRoute.to, params });
  },
});

const tenantTasksWorkflowRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tasks/$workflow',
  loader: ({ params }) => {
    throw redirect({
      to: appRoutes.tenantWorkflowRoute.to,
      params,
    });
  },
});

// redirects for alerting - redirect old non-tenanted routes to tenanted routes
// super janky using `any` since this breaks the types otherwise, since the routes
// that might be landed on don't actually exist anymore in the route tree
const workflowRunRedirectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: 'workflow-runs/$run',
  loader: ({ location, params }) => {
    const tenantId: string | null | undefined =
      (location.search as any)?.tenantId || (location.search as any)?.tenant;

    const run: string | null | undefined = (params as any)?.run;

    if (!tenantId || !run || !validate(run)) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }

    throw redirect({
      to: appRoutes.tenantRunRoute.to,
      params: { tenant: tenantId, run },
    });
  },
});

const tenantSettingsRedirect = createRoute({
  getParentRoute: () => rootRoute,
  path: 'tenant-settings',
  loader: ({ location }) => {
    const tenantId: string | null | undefined =
      (location.search as any)?.tenantId || (location.search as any)?.tenant;

    if (!tenantId) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }

    throw redirect({
      to: appRoutes.tenantSettingsOverviewRoute.to,
      params: { tenant: tenantId },
    });
  },
});

const tenantSettingsSubpathRedirect = createRoute({
  getParentRoute: () => rootRoute,
  path: 'tenant-settings/$',
  loader: ({ params, location }) => {
    const tenantId: string | null | undefined =
      (location.search as any)?.tenantId || (location.search as any)?.tenant;

    const subpath: string | null | undefined = (params as any)?._splat || '';
    const allowedSubpaths = [
      tenantSettingsAlertingRoute.path,
      tenantSettingsApiTokensRoute.path,
      tenantSettingsBillingRoute.path,
      tenantSettingsGithubRoute.path,
      tenantSettingsIngestorsRoute.path,
      tenantSettingsMembersRoute.path,
      tenantSettingsOverviewRoute.path,
    ].map((p) => p.split('/').pop());

    if (!tenantId || !subpath || !allowedSubpaths.includes(subpath)) {
      throw redirect({ to: appRoutes.authenticatedRoute.to });
    }

    throw redirect({
      to: `/tenants/${tenantId}/tenant-settings/${subpath}`,
    } as any);
  },
});

const tenantRoutes = [
  tenantEventsRoute,
  tenantFiltersRoute,
  tenantWebhooksRoute,
  tenantRateLimitsRoute,
  tenantScheduledRoute,
  tenantCronJobsRoute,
  tenantWorkflowsRoute,
  tenantWorkflowRoute,
  tenantOverviewRoute,
  tenantRunsRoute,
  tenantRunRoute,
  tenantTaskRunsRoute,
  tenantWorkersRoute,
  tenantWorkersAllRoute,
  tenantWorkerRoute,
  tenantManagedWorkersRoute,
  tenantManagedWorkersTemplateRoute,
  tenantManagedWorkersCreateRoute,
  tenantManagedWorkerRoute,
  tenantSettingsIndexRoute,
  tenantSettingsOverviewRoute,
  tenantSettingsApiTokensRoute,
  tenantSettingsGithubRoute,
  tenantSettingsMembersRoute,
  tenantSettingsAlertingRoute,
  tenantSettingsBillingRoute,
  tenantSettingsIngestorsRoute,
  tenantWorkflowRunsRedirectRoute,
  tenantWorkflowRunRedirectRoute,
  tenantTasksRedirectRoute,
  tenantTasksWorkflowRedirectRoute,
];

const routeTree = rootRoute.addChildren([
  authRoute.addChildren([authLoginRoute, authRegisterRoute]),
  onboardingVerifyRoute,
  authenticatedRoute.addChildren([
    onboardingCreateTenantRoute,
    onboardingCreateOrganizationRoute,
    onboardingInvitesRoute,
    organizationsRoute,
    organizationsNewRoute,
    tenantRoute.addChildren([tenantIndexRedirectRoute, ...tenantRoutes]),
  ]),
  v1RedirectRoute,
  workflowRunRedirectRoute,
  tenantSettingsRedirect,
  tenantSettingsSubpathRedirect,
]);

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export const appRoutes = {
  rootRoute,
  authRoute,
  authLoginRoute,
  authRegisterRoute,
  onboardingVerifyRoute,
  organizationsRoute,
  organizationsNewRoute,
  authenticatedRoute,
  onboardingCreateTenantRoute,
  onboardingCreateOrganizationRoute,
  onboardingInvitesRoute,
  tenantRoute,
  tenantEventsRoute,
  tenantFiltersRoute,
  tenantWebhooksRoute,
  tenantRateLimitsRoute,
  tenantScheduledRoute,
  tenantCronJobsRoute,
  tenantWorkflowsRoute,
  tenantWorkflowRoute,
  tenantOverviewRoute,
  tenantRunsRoute,
  tenantRunRoute,
  tenantTaskRunsRoute,
  tenantWorkersRoute,
  tenantWorkersAllRoute,
  tenantWorkerRoute,
  tenantManagedWorkersRoute,
  tenantManagedWorkersTemplateRoute,
  tenantManagedWorkersCreateRoute,
  tenantManagedWorkerRoute,
  tenantSettingsIndexRoute,
  tenantSettingsOverviewRoute,
  tenantSettingsApiTokensRoute,
  tenantSettingsGithubRoute,
  tenantSettingsMembersRoute,
  tenantSettingsAlertingRoute,
  tenantSettingsBillingRoute,
  tenantSettingsIngestorsRoute,
  tenantWorkflowRunsRedirectRoute,
  tenantWorkflowRunRedirectRoute,
  tenantTasksRedirectRoute,
  tenantTasksWorkflowRedirectRoute,
  workflowRunRedirectRoute,
};

const Router: FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
