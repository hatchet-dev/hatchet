import { FEATURES_BASE_PATH } from '@/next/lib/routes';

export const scheduledRunsRoutes = [
  {
    path: FEATURES_BASE_PATH.scheduled,
    lazy: () =>
      import('./scheduled-runs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
