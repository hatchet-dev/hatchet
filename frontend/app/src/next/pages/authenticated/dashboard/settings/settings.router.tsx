import { Navigate, RouteObject } from 'react-router-dom';

export const settingsRoutes: RouteObject[] = [
  {
    path: 'settings',
    Component: () => <Navigate to="overview" />,
  },
  {
    path: 'settings/overview',
    lazy: async () =>
      import('./overview/overview.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/api-tokens',
    lazy: async () =>
      import('./api-tokens/api-tokens.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/github',
    lazy: async () =>
      import('./github/github.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/team',
    lazy: async () =>
      import('./team/team.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/resource-limits',
    lazy: async () =>
      import('./resource-limits/resource-limits.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/alerting',
    lazy: async () =>
      import('./alerting/alerting.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'settings/ingestors',
    lazy: async () =>
      import('./ingestors/ingestors.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
