import { ROUTES } from '@/next/lib/routes';

export const workerServicesRoutes = [
  {
    path: ROUTES.services.list,
    lazy: () =>
      import('./worker-services.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.detail(':serviceName'),
    lazy: () =>
      import('./worker-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.workerDetail(':serviceName', ':workerId'),
    lazy: () =>
      import('./worker-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
