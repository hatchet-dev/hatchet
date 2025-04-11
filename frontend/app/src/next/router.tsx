import { FC } from 'react';
import {
  createBrowserRouter,
  RouteObject,
  RouterProvider,
} from 'react-router-dom';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.layout.tsx';
import { authRoutes } from './pages/auth/auth.router.tsx';
import { authenticatedRoutes } from './pages/authenticated/authenticated.router.tsx';
import { BASE_PATH } from './lib/routes';

export const routes: RouteObject[] = [
  {
    path: BASE_PATH,
    element: <Root />,
    errorElement: (
      <Root>
        <ErrorBoundary />
      </Root>
    ),
    children: [...authRoutes, ...authenticatedRoutes],
  },
];

const router = createBrowserRouter(routes, { basename: BASE_PATH });

const Router: FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
