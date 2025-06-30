import { FC } from 'react';
import {
  createBrowserRouter,
  redirect,
  RouteObject,
  RouterProvider,
} from 'react-router-dom';
import ErrorBoundary from './pages/error/index.tsx';
import Root from './pages/root.tsx';

import { routes as nextRoutes } from './next/router';

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
      ...nextRoutes,
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
        path: '/tenants/:tenant',
        lazy: async () =>
          import('./pages/authenticated').then((res) => {
            return {
              Component: res.default,
            };
          }),
        children: [
          {
            path: '/tenants/:tenant/events',
            lazy: async () =>
              import('./pages/main/v1/events').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/rate-limits',
            lazy: async () =>
              import('./pages/main/v1/rate-limits').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/scheduled',
            lazy: async () =>
              import('./pages/main/v1/scheduled-runs').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/cron-jobs',
            lazy: async () =>
              import('./pages/main/v1/recurring').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tasks',
            lazy: async () =>
              import('./pages/main/v1/workflows').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tasks/:workflow',
            lazy: async () =>
              import('./pages/main/v1/workflows/$workflow').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/runs',
            lazy: async () =>
              import('./pages/main/v1/workflow-runs-v1/index.tsx').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/runs/:run',
            lazy: async () =>
              import('./pages/main/v1/workflow-runs-v1/$run').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            // TODO: Fix this - just use the same `/runs` page
            path: '/tenants/:tenant/task-runs/:run',
            lazy: async () =>
              import('./pages/main/v1/task-runs-v1/$run').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/workers',
            lazy: async () => {
              return {
                loader: function () {
                  return redirect('/v1/workers/all');
                },
              };
            },
          },
          {
            path: '/tenants/:tenant/workers/all',
            lazy: async () =>
              import('./pages/main/v1/workers').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/workers/webhook',
            lazy: async () =>
              import('./pages/main/v1/workers/webhooks/index.tsx').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/workers/:worker',
            lazy: async () =>
              import('./pages/main/v1/workers/$worker').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/managed-workers',
            lazy: async () =>
              import('./pages/main/v1/managed-workers/index.tsx').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/managed-workers/demo-template',
            lazy: async () =>
              import(
                './pages/main/v1/managed-workers/demo-template/index.tsx'
              ).then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/managed-workers/create',
            lazy: async () =>
              import('./pages/main/v1/managed-workers/create/index.tsx').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/managed-workers/:managed-worker',
            lazy: async () =>
              import(
                './pages/main/v1/managed-workers/$managed-worker/index.tsx'
              ).then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tenant-settings/overview',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/overview').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tenant-settings/api-tokens',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/api-tokens').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/tenant-settings/github',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/github').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tenant-settings/members',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/members').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tenant-settings/alerting',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/alerting').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/tenants/:tenant/tenant-settings/billing-and-limits',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/resource-limits').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
          {
            path: '/tenants/:tenant/tenant-settings/ingestors',
            lazy: async () =>
              import('./pages/main/v1/tenant-settings/ingestors').then(
                (res) => {
                  return {
                    Component: res.default,
                  };
                },
              ),
          },
        ],
      },

      {
        path: '/v1/',
        lazy: async () =>
          import('./pages/authenticated').then((res) => {
            return {
              Component: res.default,
            };
          }),
        children: [
          {
            path: '/v1/',
            lazy: async () => {
              return {
                loader: function () {
                  return redirect('/v1/runs');
                },
              };
            },
          },
          {
            path: '/v1/workflow-runs',
            // FIXME: i'm not sure why we're still redirecting from root to here
            lazy: async () => {
              return {
                loader: function () {
                  return redirect('/v1/runs');
                },
              };
            },
          },
          {
            path: '/v1/onboarding/get-started',
            lazy: async () =>
              import('./pages/onboarding/get-started').then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: '/v1/',
            lazy: async () =>
              import('./pages/main/v1').then((res) => {
                return {
                  Component: res.default,
                };
              }),
            children: [
              {
                path: '/v1/events',
                lazy: async () =>
                  import('./pages/main/v1/events').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/rate-limits',
                lazy: async () =>
                  import('./pages/main/v1/rate-limits').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/scheduled',
                lazy: async () =>
                  import('./pages/main/v1/scheduled-runs').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/cron-jobs',
                lazy: async () =>
                  import('./pages/main/v1/recurring').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/workflows',
                // FIXME: i'm not sure why we're still redirecting from root to here
                lazy: async () => {
                  return {
                    loader: function () {
                      return redirect('/v1/tasks');
                    },
                  };
                },
              },
              {
                path: '/v1/tasks',
                lazy: async () =>
                  import('./pages/main/v1/workflows').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/workflows/:workflow',
                // FIXME: i'm not sure why we're still redirecting from root to here
                lazy: async () => {
                  return {
                    loader: function ({ params }) {
                      return redirect(`/v1/tasks/${params.workflow}`);
                    },
                  };
                },
              },
              {
                path: '/v1/tasks/:workflow',
                lazy: async () =>
                  import('./pages/main/v1/workflows/$workflow').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/runs',
                lazy: async () =>
                  import('./pages/main/v1/workflow-runs-v1/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/workflow-runs/:run',
                // FIXME: i'm not sure why we're still redirecting from root to here
                lazy: async () => {
                  return {
                    loader: function ({ params }) {
                      return redirect(`/v1/runs/${params.run}`);
                    },
                  };
                },
              },
              {
                path: '/v1/runs/:run',
                lazy: async () =>
                  import('./pages/main/v1/workflow-runs-v1/$run').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/task-runs/:run',
                lazy: async () =>
                  import('./pages/main/v1/task-runs-v1/$run').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/workers',
                lazy: async () => {
                  return {
                    loader: function () {
                      return redirect('/v1/workers/all');
                    },
                  };
                },
              },
              {
                path: '/v1/workers/all',
                lazy: async () =>
                  import('./pages/main/v1/workers').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/workers/webhook',
                lazy: async () =>
                  import('./pages/main/v1/workers/webhooks/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/workers/:worker',
                lazy: async () =>
                  import('./pages/main/v1/workers/$worker').then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/managed-workers',
                lazy: async () =>
                  import('./pages/main/v1/managed-workers/index.tsx').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/managed-workers/demo-template',
                lazy: async () =>
                  import(
                    './pages/main/v1/managed-workers/demo-template/index.tsx'
                  ).then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/managed-workers/create',
                lazy: async () =>
                  import(
                    './pages/main/v1/managed-workers/create/index.tsx'
                  ).then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/managed-workers/:managed-worker',
                lazy: async () =>
                  import(
                    './pages/main/v1/managed-workers/$managed-worker/index.tsx'
                  ).then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/tenant-settings/overview',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/overview').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/tenant-settings/api-tokens',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/api-tokens').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/tenant-settings/github',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/github').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/tenant-settings/members',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/members').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/tenant-settings/alerting',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/alerting').then(
                    (res) => {
                      return {
                        Component: res.default,
                      };
                    },
                  ),
              },
              {
                path: '/v1/tenant-settings/billing-and-limits',
                lazy: async () =>
                  import(
                    './pages/main/v1/tenant-settings/resource-limits'
                  ).then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: '/v1/tenant-settings/ingestors',
                lazy: async () =>
                  import('./pages/main/v1/tenant-settings/ingestors').then(
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
