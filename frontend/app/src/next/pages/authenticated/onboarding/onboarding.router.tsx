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
          import('./create-tenant/index').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: ROUTES.onboarding.invites,
        lazy: () =>
          import('./invites/index').then((res) => {
            return {
              Component: res.default,
              loader: res.loader,
            };
          }),
      },
      {
        path: ROUTES.onboarding.getStarted(':tenantId'),
        lazy: async () =>
          import('./get-started/index').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
    ],
  },
];
