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

export const routes: RouteObject[] = [
  {
    path: '/next',
    element: <Root />,
    errorElement: (
      <Root>
        <ErrorBoundary />
      </Root>
    ),
    children: [...authRoutes, ...authenticatedRoutes],
  },
];

const router = createBrowserRouter(routes, { basename: '/' });

const Router: FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
