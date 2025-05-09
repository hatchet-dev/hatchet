import { FEATURES_BASE_PATH, ROUTES } from '@/next/lib/routes';

export const workflowRoutes = [
  {
    path: FEATURES_BASE_PATH.workflows,
    lazy: () =>
      import('./workflows.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
  {
    path: ROUTES.workflows.detail(':workflowId'),
    lazy: () =>
      import('./workflows-detail.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
