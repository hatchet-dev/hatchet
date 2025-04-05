import { RouteObject } from 'react-router-dom';

export const scheduledRunsRoutes: RouteObject[] = [
  {
    path: '/scheduled-runs',
    lazy: async () =>
      import('./scheduled-runs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
