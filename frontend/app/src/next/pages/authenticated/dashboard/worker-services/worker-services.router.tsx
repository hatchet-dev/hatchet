import { ROUTES } from '@/next/lib/routes';
import { RouteObject } from 'react-router-dom';
import { WorkerType } from '@/lib/api';
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
    path: ROUTES.services.new(WorkerType.SELFHOSTED),
    lazy: () =>
      import('./selfhost/new-selfhost-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.new(WorkerType.MANAGED),
    lazy: () =>
      import('./managed/new-managed-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.detail(':serviceName', WorkerType.SELFHOSTED),
    lazy: () =>
      import('./selfhost/worker-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.workerDetail(
      ':serviceName',
      ':workerName',
      WorkerType.SELFHOSTED,
    ),
    lazy: () =>
      import('./selfhost/worker-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },

  {
    path: ROUTES.services.detail(':serviceName', WorkerType.MANAGED),
    lazy: () =>
      import('./managed/managed-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.workerDetail(
      ':serviceName',
      ':workerName',
      WorkerType.MANAGED,
    ),
    lazy: () =>
      import('./managed/managed-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
