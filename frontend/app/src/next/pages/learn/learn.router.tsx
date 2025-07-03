import { RouteObject } from 'react-router-dom';
import { LearnLayout } from './learn.layout';
import { FEATURES_BASE_PATH, ROUTES } from '@/next/lib/routes';

export const learnRoutes: RouteObject[] = [
  {
    path: FEATURES_BASE_PATH.learn(':tenantId'),
    element: <LearnLayout />,
    children: [
      {
        path: ROUTES.learn.firstRun(':tenantId'),
        lazy: () =>
          import('./pages/1.first-run/first-run.page').then((module) => ({
            element: <module.default />,
          })),
      },
    ],
  },
];
