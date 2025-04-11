import { RouteObject } from 'react-router-dom';
import { OnboardingLayout } from './onboarding.layout';
export const onboardingRoutes: RouteObject[] = [
  {
    path: 'onboarding',
    element: <OnboardingLayout />,
    children: [
      {
        path: 'new',
        lazy: () =>
          import('./new/new.page').then((module) => ({
            element: <module.default />,
          })),
      },
      {
        path: 'invites',
        lazy: () =>
          import('./invites/invites.page').then((module) => ({
            element: <module.default />,
          })),
      },
    ],
  },
];
