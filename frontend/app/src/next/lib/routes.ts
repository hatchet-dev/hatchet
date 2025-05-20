import { V1WorkflowRun, WorkerType } from '@/lib/api';
import { SHEET_PARAM_KEY, encodeSheetProps } from '@/next/utils/sheet-url';
import { OpenSheetProps } from '../hooks/use-side-sheet';

// Base paths
export const BASE_PATH = '/next';

export const FEATURES_BASE_PATH = {
  auth: BASE_PATH + '/auth',
  onboarding: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/onboarding`,
  learn: BASE_PATH + '/learn',
  runs: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/runs`,
  scheduled: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/scheduled`,
  crons: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/crons`,
  workflows: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/workflows`,
  services: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/services`,
  rateLimits: (tenantId: string) => BASE_PATH + `/tenants/${tenantId}/rate-limits`,
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
  },
  events: {
    list: (tenantId: string) => `${FB.events(tenantId)}`,
    detail: (tenantId: string, externalId: string) => `${FB.events(tenantId)}/${externalId}`,
  },
  learn: {
    firstRun: `${FB.learn}/first-run`,
  },
  runs: {
    list: (tenantId: string) => `${FB.runs(tenantId)}`,
    detail: (tenantId: string, runId: string) => `${FB.runs(tenantId)}/${runId}`,
    detailWithSheet: (
      tenantId: string,
      runId: string,
      sheet: OpenSheetProps,
      options?: { taskTab?: string },
    ) => {
      const params = new URLSearchParams();
      if (options?.taskTab) {
        params.set('taskTab', options.taskTab);
      }

      params.set(SHEET_PARAM_KEY, encodeSheetProps(sheet));
      return `${FB.runs(tenantId)}/${runId}?${params.toString()}`;
    },
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
    detail: (tenantId: string, workflowId: string) => `${FB.workflows(tenantId)}/${workflowId}`,
  },
  services: {
    list: (tenantId: string) => `${FB.services(tenantId)}`,
    new: (tenantId: string, type: WorkerType) => `${FB.services(tenantId)}/${type.toLowerCase()}`,
    detail: (tenantId: string, serviceName: string, type: WorkerType, tab?: string) =>
      `${FB.services(tenantId)}/${type.toLowerCase()}/${serviceName}${
        tab ? `?tab=${tab}` : ''
      }`,
    workerDetail: (tenantId: string, serviceName: string, workerName: string, type: WorkerType) =>
      `${FB.services(tenantId)}/${type.toLowerCase()}/${serviceName}/${workerName}`,
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
