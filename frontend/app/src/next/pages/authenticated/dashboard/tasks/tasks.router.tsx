import { RouteObject } from 'react-router-dom';

export const tasksRoutes: RouteObject[] = [
  {
    path: 'tasks',
    lazy: async () =>
      import('./tasks.page').then((res) => {
        return {
          Component: res.default,
        };
      }),
  },
];
