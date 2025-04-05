import { RouteObject } from 'react-router-dom';

export const workersRoutes: RouteObject[] = [
  {
    path: '/workers',
    lazy: async () =>
      import('./workers.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
