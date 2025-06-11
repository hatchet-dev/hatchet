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
          import('../../../../pages/onboarding/create-tenant').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: ROUTES.onboarding.invites,
        lazy: () =>
          import('../../../../pages/onboarding/invites').then((res) => {
            return {
              Component: res.default,
              loader: res.loader,
            };
          }),
      },
      {
        path: ROUTES.onboarding.getStarted,
        lazy: async () =>
          import('../../../../pages/onboarding/get-started').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
    ],
  },
];
