import { V1WorkflowRun, WorkerType } from '@/lib/api';

// Base paths
export const BASE_PATH = '/next';

export const FEATURES_BASE_PATH = {
  auth: BASE_PATH + '/auth',
  onboarding: BASE_PATH + `/onboarding`,
  learn: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/learn`,
  runs: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/runs`,
  scheduled: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/scheduled`,
  crons: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/crons`,
  workflows: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/workflows`,
  workers: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/workers`,
  rateLimits: (tenantId: string) =>
    BASE_PATH + `/tenants/${tenantId}/rate-limits`,
  settings: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/settings`,
  events: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/events`,
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
    newTenant: `${FB.onboarding}/create-tenant`,
    invites: `${FB.onboarding}/invites`,
    getStarted: (tenantId: string) => `${FB.onboarding}/tenants/${tenantId}/get-started`,
  },
  events: {
    list: (tenantId: string) => `${FB.events(tenantId)}`,
  },
  learn: {
    firstRun: (tenantId: string) => `${FB.learn(tenantId)}/first-run`,
  },
  runs: {
    list: (tenantId: string) => `${FB.runs(tenantId)}`,
    detail: (tenantId: string, runId: string) =>
      `${FB.runs(tenantId)}/${runId}`,
    parent: (tenantId: string, run: V1WorkflowRun) =>
      run.parentTaskExternalId
        ? `${FB.runs(tenantId)}/${run.parentTaskExternalId}`
        : undefined,
  },
  scheduled: {
    list: (tenantId: string) => `${FB.scheduled(tenantId)}`,
  },
  crons: {
    list: (tenantId: string) => `${FB.crons(tenantId)}`,
  },
  workflows: {
    list: (tenantId: string) => `${FB.workflows(tenantId)}`,
    detail: (tenantId: string, workflowId: string) =>
      `${FB.workflows(tenantId)}/${workflowId}`,
  },
  workers: {
    list: (tenantId: string) => `${FB.workers(tenantId)}`,
    new: (tenantId: string, type: WorkerType) =>
      `${FB.workers(tenantId)}/${type.toLowerCase()}`,
    poolDetail: (
      tenantId: string,
      poolName: string,
      type: WorkerType,
      tab?: string,
    ) =>
      `${FB.workers(tenantId)}/${type.toLowerCase()}/${poolName}${
        tab ? `?tab=${tab}` : ''
      }`,
    workerDetail: (
      tenantId: string,
      poolName: string,
      workerId: string,
      type: WorkerType,
    ) =>
      `${FB.workers(tenantId)}/${type.toLowerCase()}/${poolName}/${workerId}`,
  },
  rateLimits: {
    list: (tenantId: string) => `${FB.rateLimits(tenantId)}`,
  },
  settings: {
    apiTokens: (tenantId: string) => `${FB.settings(tenantId)}/api-tokens`,
    team: (tenantId: string) => `${FB.settings(tenantId)}/team`,
    overview: (tenantId: string) => `${FB.settings(tenantId)}/overview`,
    github: (tenantId: string) => `${FB.settings(tenantId)}/github`,
    usage: (tenantId: string) => `${FB.settings(tenantId)}/usage`,
    alerting: (tenantId: string) => `${FB.settings(tenantId)}/alerting`,
    ingestors: (tenantId: string) => `${FB.settings(tenantId)}/ingestors`,
  },
  common: {
    community: `https://hatchet.run/discord`,
    feedback: `https://github.com/hatchet-dev/hatchet/issues`,
    pricing: `https://hatchet.run/pricing`,
    contact: `https://hatchet.run/office-hours`,
  },
} as const;
