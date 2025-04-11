import { RouteObject } from 'react-router-dom';

export const cronJobsRoutes: RouteObject[] = [
  {
    path: 'crons',
    lazy: async () =>
      import('./cron-jobs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
