import { V1WorkflowRun, WorkerType } from '@/lib/api';

// Base paths
export const BASE_PATH = '/next';

export const FEATURES_BASE_PATH = {
  auth: BASE_PATH + '/auth',
  onboarding: BASE_PATH + '/onboarding',
  runs: BASE_PATH + '/runs',
  scheduled: BASE_PATH + '/scheduled',
  crons: BASE_PATH + '/crons',
  workflows: BASE_PATH + '/workflows',
  services: BASE_PATH + '/services',
  rateLimits: BASE_PATH + '/rate-limits',
  settings: BASE_PATH + '/settings',
};

const FB = FEATURES_BASE_PATH;

// Route paths
export const ROUTES = {
  auth: {
    login: `${FB.auth}/login`,
    register: `${FB.auth}/register`,
    verifyEmail: `${FB.auth}/verify-email`,
  },
  onboarding: {
    newTenant: `${FB.onboarding}/new`,
    invites: `${FB.onboarding}/invites`,
  },
  runs: {
    list: `${FB.runs}`,
    detail: (runId: string) => `${FB.runs}/${runId}`,
    taskDetail: (
      runId: string,
      taskId: string,
      options?: { task_tab?: string },
    ) =>
      `${FB.runs}/${runId}/${taskId}${
        options ? `?${new URLSearchParams(options).toString()}` : ''
      }`,
    parent: (run: V1WorkflowRun) =>
      run.parentTaskExternalId
        ? `${FB.runs}/${run.parentTaskExternalId}`
        : undefined,
  },
  scheduled: {
    list: `${FB.scheduled}`,
  },
  crons: {
    list: `${FB.crons}`,
  },
  workflows: {
    list: `${FB.workflows}`,
    detail: (workflowId: string) => `${FB.workflows}/${workflowId}`,
  },
  services: {
    list: `${FB.services}`,
    new: (type: WorkerType) => `${FB.services}/${type.toLowerCase()}`,
    detail: (serviceName: string, type: WorkerType) =>
      `${FB.services}/${type.toLowerCase()}/${serviceName}`,
    workerDetail: (serviceName: string, workerName: string, type: WorkerType) =>
      `${FB.services}/${type.toLowerCase()}/${serviceName}/${workerName}`,
  },
  rateLimits: {
    list: `${FB.rateLimits}`,
  },
  settings: {
    apiTokens: `${FB.settings}/api-tokens`,
    team: `${FB.settings}/team`,
    overview: `${FB.settings}/overview`,
    github: `${FB.settings}/github`,
    usage: `${FB.settings}/usage`,
    alerting: `${FB.settings}/alerting`,
    ingestors: `${FB.settings}/ingestors`,
  },
  common: {
    community: `https://hatchet.run/discord`,
    feedback: `https://github.com/hatchet-dev/hatchet/issues`,
    pricing: `https://hatchet.run/pricing`,
    tutorial: `${BASE_PATH}/tutorial`,
    contact: `https://hatchet.run/office-hours`,
  },
} as const;

// Type for route paths
export type RoutePath = (typeof ROUTES)[keyof typeof ROUTES] extends string
  ? string
  : never;
