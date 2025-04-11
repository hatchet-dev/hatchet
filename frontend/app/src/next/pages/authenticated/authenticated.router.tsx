import { RouteObject } from 'react-router-dom';
import AuthenticatedGuard from './authenticated.guard.tsx';
import { dashboardRoutes } from './dashboard/dashboard.router.tsx';
import { onboardingRoutes } from './onboarding/onboarding.router.tsx';

export const authenticatedRoutes: RouteObject[] = [
  {
    path: '/',
    element: <AuthenticatedGuard />,
    children: [...dashboardRoutes, ...onboardingRoutes],
  },
];
