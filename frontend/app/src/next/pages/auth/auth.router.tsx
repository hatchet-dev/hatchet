import { RouteObject } from 'react-router-dom';
import { ROUTES, FEATURES_BASE_PATH } from '@/next/lib/routes';

export const authRoutes: RouteObject[] = [
  {
    path: FEATURES_BASE_PATH.auth,
    lazy: async () =>
      import('./auth.layout').then((res) => {
        return {
          Component: res.default,
        };
      }),
    children: [
      {
        path: ROUTES.auth.login,
        lazy: async () =>
          import('./login/login.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: ROUTES.auth.register,
        lazy: async () =>
          import('./register/register.page').then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: ROUTES.auth.verifyEmail,
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
