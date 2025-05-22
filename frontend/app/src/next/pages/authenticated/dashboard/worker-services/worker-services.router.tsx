import { ROUTES } from '@/next/lib/routes';
import { RouteObject } from 'react-router-dom';
import { WorkerType } from '@/lib/api';
export const workerRoutes: RouteObject[] = [
  {
    path: ROUTES.services.list(':tenantId'),
    lazy: () =>
      import('./worker-services.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.new(':tenantId', WorkerType.SELFHOSTED),
    lazy: () =>
      import('./selfhost/new-selfhost-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.new(':tenantId', WorkerType.MANAGED),
    lazy: () =>
      import('./managed/new-managed-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.detail(
      ':tenantId',
      ':serviceName',
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
    path: ROUTES.services.workerDetail(
      ':tenantId',
      ':serviceName',
      ':workerId',
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
    path: ROUTES.services.detail(
      ':tenantId',
      ':serviceName',
      WorkerType.MANAGED,
    ),
    lazy: () =>
      import('./managed/managed-service-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.services.workerDetail(
      ':tenantId',
      ':serviceName',
      ':workerId',
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
