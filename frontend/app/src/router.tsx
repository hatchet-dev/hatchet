import React from "react";
import { createBrowserRouter, RouterProvider } from "react-router-dom";

const routes = [
  {
    path: "/",
    lazy: async () =>
      import("./pages/root.tsx").then((res) => {
        return {
          Component: res.default,
        };
      }),
    children: [
      {
        path: "/auth/login",
        lazy: async () =>
          import("./pages/auth/login").then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: "/auth/register",
        lazy: async () =>
          import("./pages/auth/register").then((res) => {
            return {
              Component: res.default,
            };
          }),
      },
      {
        path: "/",
        lazy: async () =>
          import("./pages/main/auth").then((res) => {
            return {
              loader: res.loader,
              Component: res.default,
            };
          }),
        children: [
          {
            path: "/onboarding/create-tenant",
            lazy: async () =>
              import("./pages/onboarding/create-tenant").then((res) => {
                return {
                  Component: res.default,
                };
              }),
          },
          {
            path: "/",
            lazy: async () =>
              import("./pages/main").then((res) => {
                return {
                  Component: res.default,
                };
              }),
            children: [
              {
                path: "/events",
                lazy: async () =>
                  import("./pages/main/events").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/events/metrics",
                lazy: async () =>
                  import("./pages/main/events/metrics").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/workflows",
                lazy: async () =>
                  import("./pages/main/workflows").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/workflows/:workflow",
                lazy: async () =>
                  import("./pages/main/workflows/$workflow").then((res) => {
                    return {
                      Component: res.default,
                      loader: res.loader,
                    };
                  }),
              },
              {
                path: "/workflow-runs",
                lazy: async () =>
                  import("./pages/main/workflow-runs").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/workflow-runs/:run",
                lazy: async () =>
                  import("./pages/main/workflow-runs/$run").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/workers",
                lazy: async () =>
                  import("./pages/main/workers").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
              {
                path: "/workers/:worker",
                lazy: async () =>
                  import("./pages/main/workers/$worker").then((res) => {
                    return {
                      Component: res.default,
                    };
                  }),
              },
            ],
          },
        ],
      },
    ],
  },
];

const router = createBrowserRouter(routes, { basename: "/" });

const Router: React.FC = () => {
  return <RouterProvider router={router} />;
};

export default Router;
