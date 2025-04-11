import { FEATURES_BASE_PATH } from '@/next/lib/routes';

export const tasksRoutes = [
  {
    path: FEATURES_BASE_PATH.tasks,
    lazy: () =>
      import('./tasks.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
