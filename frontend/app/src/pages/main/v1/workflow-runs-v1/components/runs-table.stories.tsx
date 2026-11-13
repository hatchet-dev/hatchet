import { RunsProvider } from '../hooks/runs-provider';
import { RunsTable } from './runs-table';
import {
  loadingHandlers,
  runsHandlers,
  STORY_TENANT_ID,
} from './runs-table.fixtures';
import { RefetchIntervalProvider } from '@/contexts/refetch-interval-context';
import { SidePanelProvider } from '@/hooks/use-side-panel';
import { AppContextProvider } from '@/providers/app-context';
import { UserUniverseProvider } from '@/providers/user-universe';
import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router';
import { ReactNode, useState } from 'react';

// RunsTable reads the tenant from the route and keeps its filters in the URL,
// so it runs inside a real router on an in-memory history.
function RunsPageHarness({
  children,
  width,
}: {
  children: ReactNode;
  width?: number;
}) {
  // A client per story, so one variant's cached runs never show in another.
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: false } },
      }),
  );
  const [router] = useState(() => {
    const rootRoute = createRootRoute();
    const runsRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/tenants/$tenant/runs',
      component: () => (
        <UserUniverseProvider>
          <AppContextProvider>
            <SidePanelProvider>
              <RefetchIntervalProvider>
                <div className="h-[860px]" style={{ width }}>
                  {children}
                </div>
              </RefetchIntervalProvider>
            </SidePanelProvider>
          </AppContextProvider>
        </UserUniverseProvider>
      ),
    });

    return createRouter({
      routeTree: rootRoute.addChildren([runsRoute]),
      history: createMemoryHistory({
        initialEntries: [`/tenants/${STORY_TENANT_ID}/runs`],
      }),
    });
  });

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}

const withRunsPage: Decorator = (Story, context) => (
  <RunsPageHarness width={context.parameters.harnessWidth}>
    <Story />
  </RunsPageHarness>
);

const meta = {
  title: 'Pages/Runs/RunsTable',
  component: RunsTable,
  decorators: [withRunsPage],
  render: () => (
    <RunsProvider tableKey="workflow-runs-story">
      <RunsTable />
    </RunsProvider>
  ),
  parameters: {
    msw: { handlers: runsHandlers(60) },
  },
} satisfies Meta<typeof RunsTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const WithRuns: Story = {};

export const Empty: Story = {
  parameters: {
    msw: { handlers: runsHandlers(0) },
  },
};

export const Loading: Story = {
  parameters: {
    msw: { handlers: loadingHandlers() },
  },
};

// Below the 900px container breakpoint the sidebar stacks above the table.
export const NarrowWidth: Story = {
  parameters: {
    harnessWidth: 760,
  },
};
