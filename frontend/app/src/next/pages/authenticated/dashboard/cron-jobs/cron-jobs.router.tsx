import { RouteObject } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
export const cronJobsRoutes: RouteObject[] = [
  {
    path: ROUTES.crons.list,
    lazy: async () =>
      import('./cron-jobs.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
