import { RouteObject } from 'react-router-dom';
import Authenticated from './authenticated.layout.tsx';
import { dashboardRoutes } from './dashboard/dashboard.router.tsx';
export const authenticatedRoutes: RouteObject[] = [
  {
    path: '/',
    element: <Authenticated />,
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
