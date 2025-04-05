import { RouteObject, Navigate } from 'react-router-dom';
import DashboardLayout from './dashboard.layout';
export const dashboardRoutes: RouteObject[] = [
  {
    path: '/',
    element: <DashboardLayout />,
    children: [
      {
        path: '/',
        Component: () => <Navigate to="/runs" />,
      },
      {
        path: '/runs',
        lazy: async () =>
          import('./runs/runs.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
    ],
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
