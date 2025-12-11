import { FC } from 'react';
import {
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  redirect,
} from '@tanstack/react-router';
import { Outlet } from '@tanstack/react-router';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.tsx';

const rootRoute = createRootRoute({
  component: Root,
  errorComponent: (props) => (
    <Root>
      <ErrorBoundary {...props} />
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
  component: Outlet,
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
  getParentRoute: () => rootRoute,
  path: 'organizations/$organization',
  component: lazyRouteComponent(
    () => import('./pages/organizations/$organization'),
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
});

const onboardingCreateTenantRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'onboarding/create-tenant',
  component: lazyRouteComponent(
    () => import('./pages/onboarding/create-tenant'),
    'default',
  ),
});

const onboardingGetStartedRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'onboarding/get-started',
  component: lazyRouteComponent(
    () => import('./pages/onboarding/get-started'),
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
  component: lazyRouteComponent(() => import('./pages/main/v1'), 'default'),
});

const tenantIndexRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: '/',
  loader: ({ params }) => {
    throw redirect({ to: '/tenants/$tenant/runs', params });
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
    throw redirect({ to: '/tenants/$tenant/runs/$run', params });
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
    throw redirect({ to: '/tenants/$tenant/workers', params });
  },
});

const tenantWorkersWebhookRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workers/webhook',
  component: lazyRouteComponent(
    () => import('./pages/main/v1/workers/webhooks/index.tsx'),
    'default',
  ),
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
  path: 'managed-workers/$managed-worker',
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
      to: '/tenants/$tenant/tenant-settings/overview',
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

const tenantOnboardingGetStartedRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'onboarding/get-started',
  component: lazyRouteComponent(
    () => import('./pages/onboarding/get-started'),
    'default',
  ),
});

const tenantWorkflowRunsRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflow-runs',
  loader: ({ params }) => {
    throw redirect({ to: '/tenants/$tenant/runs', params });
  },
});

const tenantWorkflowRunRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'workflow-runs/$run',
  loader: ({ params }) => {
    throw redirect({ to: '/tenants/$tenant/runs/$run', params });
  },
});

const tenantTasksRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tasks',
  loader: ({ params }) => {
    throw redirect({ to: '/tenants/$tenant/workflows', params });
  },
});

const tenantTasksWorkflowRedirectRoute = createRoute({
  getParentRoute: () => tenantRoute,
  path: 'tasks/$workflow',
  loader: ({ params }) => {
    throw redirect({
      to: '/tenants/$tenant/workflows/$workflow',
      params,
    });
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
  tenantRunsRoute,
  tenantRunRoute,
  tenantTaskRunsRoute,
  tenantWorkersRoute,
  tenantWorkersAllRoute,
  tenantWorkersWebhookRoute,
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
  tenantOnboardingGetStartedRoute,
  tenantWorkflowRunsRedirectRoute,
  tenantWorkflowRunRedirectRoute,
  tenantTasksRedirectRoute,
  tenantTasksWorkflowRedirectRoute,
];

const routeTree = rootRoute.addChildren([
  authRoute.addChildren([authLoginRoute, authRegisterRoute]),
  onboardingVerifyRoute,
  organizationsRoute,
  authenticatedRoute.addChildren([
    onboardingCreateTenantRoute,
    onboardingGetStartedRoute,
    onboardingInvitesRoute,
    tenantRoute.addChildren([tenantIndexRedirectRoute, ...tenantRoutes]),
  ]),
  v1RedirectRoute,
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
  authenticatedRoute,
  onboardingCreateTenantRoute,
  onboardingGetStartedRoute,
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
  tenantRunsRoute,
  tenantRunRoute,
  tenantTaskRunsRoute,
  tenantWorkersRoute,
  tenantWorkersAllRoute,
  tenantWorkersWebhookRoute,
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
  tenantOnboardingGetStartedRoute,
  tenantWorkflowRunsRedirectRoute,
  tenantWorkflowRunRedirectRoute,
  tenantTasksRedirectRoute,
  tenantTasksWorkflowRedirectRoute,
};

const Router: FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
