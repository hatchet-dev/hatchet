import { ROUTES } from '@/next/lib/routes';

export const rateLimitsRoutes = [
  {
    path: ROUTES.rateLimits.list(':tenantId'),
    lazy: () =>
      import('./rate-limits.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
