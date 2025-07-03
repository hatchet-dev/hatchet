import { FC } from 'react';
import {
  createBrowserRouter,
  redirect,
  RouteObject,
  RouterProvider,
} from 'react-router-dom';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.tsx';

export const tenantedPaths = [
  '/tenants/:tenant/events',
  '/tenants/:tenant/rate-limits',
  '/tenants/:tenant/scheduled',
  '/tenants/:tenant/cron-jobs',
  '/tenants/:tenant/tasks',
  '/tenants/:tenant/tasks/:workflow',
  '/tenants/:tenant/runs',
  '/tenants/:tenant/runs/:run',
  '/tenants/:tenant/task-runs/:run',
  '/tenants/:tenant/workers',
  '/tenants/:tenant/workers/all',
  '/tenants/:tenant/workers/webhook',
  '/tenants/:tenant/workers/:worker',
  '/tenants/:tenant/managed-workers',
  '/tenants/:tenant/managed-workers/demo-template',
  '/tenants/:tenant/managed-workers/create',
  '/tenants/:tenant/managed-workers/:managed-worker',
  '/tenants/:tenant/tenant-settings',
  '/tenants/:tenant/tenant-settings/overview',
  '/tenants/:tenant/tenant-settings/api-tokens',
  '/tenants/:tenant/tenant-settings/github',
  '/tenants/:tenant/tenant-settings/members',
  '/tenants/:tenant/tenant-settings/alerting',
  '/tenants/:tenant/tenant-settings/billing-and-limits',
  '/tenants/:tenant/tenant-settings/ingestors',
  '/tenants/:tenant/onboarding/get-started',
  '/tenants/:tenant/workflow-runs',
  '/tenants/:tenant/workflow-runs/:run',
  '/tenants/:tenant/',
  '/tenants/:tenant/workflows',
  '/tenants/:tenant/workflows/:workflow',
] as const;

export type TenantedPath = (typeof tenantedPaths)[number];

const createTenantedRoute = (path: TenantedPath): RouteObject => {
  switch (path) {
    case '/tenants/:tenant/events':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/events').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/rate-limits':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/rate-limits').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/scheduled':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/scheduled-runs').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/cron-jobs':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/recurring').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tasks':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workflows').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tasks/:workflow':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workflows/$workflow').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/runs':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workflow-runs-v1/index.tsx').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/runs/:run':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workflow-runs-v1/$run').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/task-runs/:run':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/task-runs-v1/$run').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/workers':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(`/tenants/${params.tenant}/workers/all`);
            },
          };
        },
      };
    case '/tenants/:tenant/workers/all':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workers').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/workers/webhook':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workers/webhooks/index.tsx').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/workers/:worker':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/workers/$worker').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/managed-workers':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/managed-workers/index.tsx').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/managed-workers/demo-template':
      return {
        path,
        lazy: async () =>
          import(
            './pages/main/v1/managed-workers/demo-template/index.tsx'
          ).then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/managed-workers/create':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/managed-workers/create/index.tsx').then(
            (res) => {
              return {
                Component: res.default,
              };
            },
          ),
      };
    case '/tenants/:tenant/managed-workers/:managed-worker':
      return {
        path,
        lazy: async () =>
          import(
            './pages/main/v1/managed-workers/$managed-worker/index.tsx'
          ).then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/overview':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/overview').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/api-tokens':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/api-tokens').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/github':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/github').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/members':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/members').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/alerting':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/alerting').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/tenant-settings/billing-and-limits':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/resource-limits').then(
            (res) => {
              return {
                Component: res.default,
              };
            },
          ),
      };
    case '/tenants/:tenant/tenant-settings/ingestors':
      return {
        path,
        lazy: async () =>
          import('./pages/main/v1/tenant-settings/ingestors').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/onboarding/get-started':
      return {
        path,
        lazy: async () =>
          import('./pages/onboarding/get-started').then((res) => {
            return {
              Component: res.default,
            };
          }),
      };
    case '/tenants/:tenant/workflow-runs':
      return {
        path,
        loader: ({ params }) => {
          return redirect(`/tenants/${params.tenant}/runs`);
        },
      };
    case '/tenants/:tenant/workflow-runs/:run':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(`/tenants/${params.tenant}/runs/${params.run}`);
            },
          };
        },
      };
    case '/tenants/:tenant/':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(`/tenants/${params.tenant}/runs`);
            },
          };
        },
      };
    case '/tenants/:tenant/workflows':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(`/tenants/${params.tenant}/tasks`);
            },
          };
        },
      };
    case '/tenants/:tenant/workflows/:workflow':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(
                `/tenants/${params.tenant}/tasks/${params.workflow}`,
              );
            },
          };
        },
      };
    case '/tenants/:tenant/tenant-settings':
      return {
        path,
        lazy: async () => {
          return {
            loader: function ({ params }) {
              return redirect(
                `/tenants/${params.tenant}/tenant-settings/overview`,
              );
            },
          };
        },
      };
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = path;
      throw new Error(`Unhandled path: ${exhaustiveCheck}`);
  }
};

export const routes: RouteObject[] = [
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
              Component: res.default,
              loader: res.loader,
            };
          }),
      },
      {
        path: '/',
        lazy: async () =>
          import('./pages/authenticated').then((res) => {
            return {
              Component: res.default,
            };
          }),
        children: [
          {
            path: '/',
            lazy: async () => {
              return {
                loader: function () {
                  return redirect('/workflow-runs');
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
                  Component: res.default,
                  loader: res.loader,
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
                path: '/rate-limits',
                lazy: async () =>
                  import('./pages/main/rate-limits').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/scheduled',
                lazy: async () =>
                  import('./pages/main/scheduled-runs').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/cron-jobs',
                lazy: async () =>
                  import('./pages/main/recurring').then((res) => {
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
                lazy: async () => {
                  return {
                    loader: function () {
                      return redirect('/workers/all');
                    },
                  };
                },
              },
              {
                path: '/workers/all',
                lazy: async () =>
                  import('./pages/main/workers').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/workers/webhook',
                lazy: async () =>
                  import('./pages/main/workers/webhooks/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
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
                path: '/managed-workers',
                lazy: async () =>
                  import('./pages/main/managed-workers/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/managed-workers/demo-template',
                lazy: async () =>
                  import(
                    './pages/main/managed-workers/demo-template/index.tsx'
                  ).then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/managed-workers/create',
                lazy: async () =>
                  import('./pages/main/managed-workers/create/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/managed-workers/:managed-worker',
                lazy: async () =>
                  import(
                    './pages/main/managed-workers/$managed-worker/index.tsx'
                  ).then((res) => {
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
                path: '/tenant-settings/github',
                lazy: async () =>
                  import('./pages/main/tenant-settings/github').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
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
      {
        path: '/v1/*',
        lazy: async () => {
          return {
            loader: function () {
              return redirect('/');
            },
          };
        },
      },
      {
        path: '/tenants/:tenant',
        lazy: async () =>
          import('./pages/authenticated').then((res) => {
            return {
              Component: res.default,
            };
          }),
        children: [
          {
            path: '/tenants/:tenant',
            lazy: async () =>
              import('./pages/main/v1').then((res) => {
                return {
                  Component: res.default,
                };
              }),
            children: tenantedPaths.map((path) => createTenantedRoute(path)),
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
