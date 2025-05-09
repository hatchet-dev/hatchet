import { ROUTES } from '@/next/lib/routes';

export const runsRoutes = [
  {
    path: ROUTES.runs.list,
    lazy: () =>
      import('./runs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.runs.detail(':workflowRunId'),
    lazy: () =>
      import('./run-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.runs.taskDetail(':workflowRunId', ':taskId'),
    lazy: () =>
      import('./run-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
