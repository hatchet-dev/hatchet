import { Navigate, RouteObject } from 'react-router-dom';
import { ROUTES, FEATURES_BASE_PATH } from '@/next/lib/routes';

export const settingsRoutes: RouteObject[] = [
  {
    path: FEATURES_BASE_PATH.settings(':tenantId'),
    Component: () => <Navigate to="overview" />,
  },
  {
    path: ROUTES.settings.overview(':tenantId'),
    lazy: async () =>
      import('./overview/overview.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.apiTokens(':tenantId'),
    lazy: async () =>
      import('./api-tokens/api-tokens.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.github(':tenantId'),
    lazy: async () =>
      import('./github/github.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.team(':tenantId'),
    lazy: async () =>
      import('./team/team.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.usage(':tenantId'),
    lazy: async () =>
      import('./usage/usage.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.alerting(':tenantId'),
    lazy: async () =>
      import('./alerting/alerting.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.ingestors(':tenantId'),
    lazy: async () =>
      import('./ingestors/ingestors.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
