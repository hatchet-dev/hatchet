import { FC } from 'react';
import {
  createBrowserRouter,
  redirect,
  RouteObject,
  RouterProvider,
} from 'react-router-dom';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.tsx';

const routes: RouteObject[] = [
  {
    path: '/',
    element: <Root />,
    errorElement: (
      <Root>
        <ErrorBoundary />
      </Root>
    ),
    children: [
      {
        path: '/auth',
        lazy: async () =>
          import('./pages/auth/no-auth').then((res) => {
            return {
              loader: res.loader,
            };
          }),
        children: [
          {
            path: '/auth/login',
            lazy: async () =>
              import('./pages/auth/login').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/auth/register',
            lazy: async () =>
              import('./pages/auth/register').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
        ],
      },
      {
        path: '/onboarding/verify-email',
        lazy: async () =>
          import('./pages/onboarding/verify-email').then((res) => {
            return {
              loader: res.loader,
              Component: res.default,
            };
          }),
      },
      {
        path: '/',
        lazy: async () =>
          import('./pages/authenticated').then((res) => {
            return {
              loader: res.loader,
              Component: res.default,
            };
          }),
        children: [
          {
            path: '/',
            lazy: async () => {
              return {
                loader: function () {
                  return redirect('/events');
                },
              };
            },
          },
          {
            path: '/onboarding/create-tenant',
            lazy: async () =>
              import('./pages/onboarding/create-tenant').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/onboarding/get-started',
            lazy: async () =>
              import('./pages/onboarding/get-started').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/onboarding/invites',
            lazy: async () =>
              import('./pages/onboarding/invites').then((res) => {
                return {
                  loader: res.loader,
                  Component: res.default,
                };
              }),
          },
          {
            path: '/',
            lazy: async () =>
              import('./pages/main').then((res) => {
                return {
                  Component: res.default,
                };
              }),
            children: [
              {
                path: '/events',
                lazy: async () =>
                  import('./pages/main/events').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/events/metrics',
                lazy: async () =>
                  import('./pages/main/events/metrics').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workflows',
                lazy: async () =>
                  import('./pages/main/workflows').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workflows/:workflow',
                lazy: async () =>
                  import('./pages/main/workflows/$workflow').then((res) => {
                    return {
                      Component: res.default,
                      loader: res.loader,
                    };
                  }),
              },
              {
                path: '/workflow-runs',
                lazy: async () =>
                  import('./pages/main/workflow-runs').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workflow-runs/:run',
                lazy: async () =>
                  import('./pages/main/workflow-runs/$run').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workers',
                lazy: async () =>
                  import('./pages/main/workers').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workers/:worker',
                lazy: async () =>
                  import('./pages/main/workers/$worker').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/tenant-settings/overview',
                lazy: async () =>
                  import('./pages/main/tenant-settings/overview').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/tenant-settings/api-tokens',
                lazy: async () =>
                  import('./pages/main/tenant-settings/api-tokens').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/tenant-settings/webhooks',
                lazy: async () =>
                  import('./pages/main/tenant-settings/webhooks').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/tenant-settings/members',
                lazy: async () =>
                  import('./pages/main/tenant-settings/members').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/tenant-settings/alerting',
                lazy: async () =>
                  import('./pages/main/tenant-settings/alerting').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/tenant-settings/billing-and-limits',
                lazy: async () =>
                  import('./pages/main/tenant-settings/resource-limits').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/tenant-settings/ingestors',
                lazy: async () =>
                  import('./pages/main/tenant-settings/ingestors').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
            ],
          },
        ],
      },
    ],
  },
];

const router = createBrowserRouter(routes, { basename: '/' });

const Router: FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
