import { RouteObject } from 'react-router-dom';

export const authRoutes: RouteObject[] = [
  {
    path: '/auth',
    lazy: async () =>
      import('./no-auth.middleware').then((res) => {
        return {
          loader: res.loader,
        };
      }),
    children: [
      {
        path: '/auth/login',
        lazy: async () =>
          import('./login').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: '/auth/register',
        lazy: async () =>
          import('./register').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
    ],
  },
];
