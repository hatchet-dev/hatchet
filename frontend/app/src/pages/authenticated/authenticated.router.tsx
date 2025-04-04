import { RouteObject } from 'react-router-dom';
import AuthenticatedGuard from './authenticated.guard.tsx';
import { dashboardRoutes } from './dashboard/dashboard.router.tsx';
import OnboardingNewPage from './onboarding/new/new.page.tsx';
export const authenticatedRoutes: RouteObject[] = [
  {
    path: '/',
    element: <AuthenticatedGuard />,
    children: [...dashboardRoutes],
  },
  {
    path: '/onboarding/new',
    element: <OnboardingNewPage />,
  },
];

// {
//     path: '/',
//     lazy: async () =>
//       import('./pages/authenticated/authenticated.tsx').then((res) => {
//         return {
//           Component: res.default,
//         };
//       }),
//   },
