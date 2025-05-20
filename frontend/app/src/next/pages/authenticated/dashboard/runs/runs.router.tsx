import { ROUTES } from '@/next/lib/routes';

export const runsRoutes = [
  {
    path: ROUTES.runs.list(':tenantId'),
    lazy: () =>
      import('./runs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.runs.detail(':tenantId', ':workflowRunId'),
    lazy: () =>
      import('./run-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
