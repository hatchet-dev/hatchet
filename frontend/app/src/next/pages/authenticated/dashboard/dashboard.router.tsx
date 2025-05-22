import { RouteObject, Navigate } from 'react-router-dom';
import DashboardLayout from './dashboard.layout';
import { settingsRoutes } from './settings/settings.router';
import { runsRoutes } from './runs/runs.router';
import { scheduledRunsRoutes } from './scheduled-runs/scheduled-runs.router';
import { cronJobsRoutes } from './cron-jobs/cron-jobs.router';
import { workflowRoutes } from './workflows/workflows.router';
import { workerRoutes } from './worker-services/worker-services.router';
import { rateLimitsRoutes } from './rate-limits/rate-limits.router';
import { eventsRoutes } from './events/events.router';

export const dashboardRoutes: RouteObject[] = [
  {
    path: '',
    element: <DashboardLayout />,
    children: [
      {
        path: '',
        Component: () => <Navigate to="runs" />,
      },
      ...runsRoutes,
      ...eventsRoutes,
      ...scheduledRunsRoutes,
      ...cronJobsRoutes,
      ...workflowRoutes,
      ...workerRoutes,
      ...rateLimitsRoutes,
      ...settingsRoutes,
    ],
  },
];
