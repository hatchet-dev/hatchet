import { RouteObject } from 'react-router-dom';
import { OnboardingLayout } from './onboarding.layout';
import { FEATURES_BASE_PATH, ROUTES } from '@/next/lib/routes';

export const onboardingRoutes: RouteObject[] = [
  {
    path: FEATURES_BASE_PATH.onboarding,
    element: <OnboardingLayout />,
    children: [
      {
        path: ROUTES.onboarding.newTenant,
        lazy: () =>
          import('./new/new.page').then((module) => ({
            element: <module.default />,
          })),
      },
      {
        path: ROUTES.onboarding.invites,
        lazy: () =>
          import('./invites/invites.page').then((module) => ({
            element: <module.default />,
          })),
      },
    ],
  },
];
