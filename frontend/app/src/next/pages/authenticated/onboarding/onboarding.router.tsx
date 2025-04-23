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
          import('./new/1.new.page').then((module) => ({
            element: <module.default />,
          })),
      },
      {
        path: ROUTES.onboarding.firstRun,
        lazy: () =>
          import('./new/2.first-run.page').then((module) => ({
            element: <module.default />,
          })),
      },
      {
        path: ROUTES.onboarding.inviteTeam,
        lazy: () =>
          import('./new/3.invite-team.page').then((module) => ({
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
