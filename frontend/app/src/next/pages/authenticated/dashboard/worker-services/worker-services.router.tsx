import { RouteObject } from 'react-router-dom';

export const workerServicesRoutes: RouteObject[] = [
  {
    path: 'services',
    lazy: async () =>
      import('./worker-services.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'services/:serviceName',
    lazy: async () =>
      import('./worker-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: 'services/:serviceName/:workerId',
    lazy: async () =>
      import('./worker-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
