import { RouteObject } from 'react-router-dom';
import AuthenticatedGuard from './authenticated.guard.tsx';
import { dashboardRoutes } from './dashboard/dashboard.router.tsx';
export const authenticatedRoutes: RouteObject[] = [
  {
    path: '/',
    element: <AuthenticatedGuard />,
    children: [...dashboardRoutes],
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
