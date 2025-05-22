import { ROUTES } from '@/next/lib/routes';
import { RouteObject } from 'react-router-dom';
import { WorkerType } from '@/lib/api';
export const workerRoutes: RouteObject[] = [
  {
    path: ROUTES.workers.list(':tenantId'),
    lazy: () =>
      import('./worker-pools.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.workers.new(':tenantId', WorkerType.SELFHOSTED),
    lazy: () =>
      import('./selfhost/new-selfhost-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.workers.new(':tenantId', WorkerType.MANAGED),
    lazy: () =>
      import('./managed/new-managed-service.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.workers.poolDetail(
      ':tenantId',
      ':poolName',
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
    path: ROUTES.workers.workerDetail(
      ':tenantId',
      ':poolName',
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
    path: ROUTES.workers.poolDetail(
      ':tenantId',
      ':poolName',
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
    path: ROUTES.workers.workerDetail(
      ':tenantId',
      ':poolName',
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
