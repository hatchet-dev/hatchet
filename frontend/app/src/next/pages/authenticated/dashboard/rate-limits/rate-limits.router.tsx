import { RouteObject } from 'react-router-dom';

export const rateLimitsRoutes: RouteObject[] = [
  {
    path: 'rate-limits',
    lazy: async () =>
      import('./rate-limits.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
