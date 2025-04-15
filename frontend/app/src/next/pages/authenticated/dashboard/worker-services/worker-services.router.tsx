import { ROUTES } from '@/next/lib/routes';
import { RouteObject } from 'react-router-dom';

export const workerServicesRoutes: RouteObject[] = [
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
      import('./worker-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
