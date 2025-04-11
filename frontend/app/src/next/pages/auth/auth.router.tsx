import { RouteObject } from 'react-router-dom';

export const authRoutes: RouteObject[] = [
  {
    path: '/auth',
    lazy: async () =>
      import('./auth.layout').then((res) => {
        return {
          Component: res.default,
        };
      }),
    children: [
      {
        path: '/auth/login',
        lazy: async () =>
          import('./login/login.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: '/auth/register',
        lazy: async () =>
          import('./register/register.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: '/auth/verify-email',
        lazy: async () =>
          import('./verify-email/verify-email.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
    ],
  },
];
