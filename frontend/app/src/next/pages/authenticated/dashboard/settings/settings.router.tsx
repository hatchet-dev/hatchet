import { Navigate, RouteObject } from 'react-router-dom';
import { ROUTES, FEATURES_BASE_PATH } from '@/next/lib/routes';

export const settingsRoutes: RouteObject[] = [
  {
    path: FEATURES_BASE_PATH.settings,
    Component: () => <Navigate to="overview" />,
  },
  {
    path: ROUTES.settings.overview,
    lazy: async () =>
      import('./overview/overview.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.apiTokens,
    lazy: async () =>
      import('./api-tokens/api-tokens.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.github,
    lazy: async () =>
      import('./github/github.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.team,
    lazy: async () =>
      import('./team/team.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.resourceLimits,
    lazy: async () =>
      import('./resource-limits/resource-limits.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.alerting,
    lazy: async () =>
      import('./alerting/alerting.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.settings.ingestors,
    lazy: async () =>
      import('./ingestors/ingestors.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
