import { ROUTES } from '@/next/lib/routes';

export const eventsRoutes = [
  {
    path: ROUTES.events.list(':tenantId'),
    lazy: () =>
      import('./events.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
