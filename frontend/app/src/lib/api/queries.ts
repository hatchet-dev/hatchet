import { createQueryKeyStore } from '@lukemorales/query-key-factory';

import api from './api';
import invariant from 'tiny-invariant';
import { PullRequestState } from '.';

type ListEventQuery = Parameters<typeof api.eventList>[1];
type ListLogLineQuery = Parameters<typeof api.logLineList>[1];
type ListWorkflowRunsQuery = Parameters<typeof api.workflowRunList>[1];
type WorkflowRunMetrics = Parameters<typeof api.workflowRunGetMetrics>[1];

export const queries = createQueryKeyStore({
  user: {
    current: {
      queryKey: ['user:get'],
      queryFn: async () => (await api.userGetCurrent()).data,
    },
    listTenantMemberships: {
      queryKey: ['tenant-memberships:list'],
      queryFn: async () => (await api.tenantMembershipsList()).data,
    },
    listInvites: {
      queryKey: ['user:list:tenant-invites'],
      queryFn: async () => (await api.userListTenantInvites()).data,
    },
  },
  members: {
    list: (tenant: string) => ({
      queryKey: ['tenant-member:list', tenant],
      queryFn: async () => (await api.tenantMemberList(tenant)).data,
    }),
  },
  tokens: {
    list: (tenant: string) => ({
      queryKey: ['api-token:list', tenant],
      queryFn: async () => (await api.apiTokenList(tenant)).data,
    }),
  },
  snsIntegrations: {
    list: (tenant: string) => ({
      queryKey: ['sns:list', tenant],
      queryFn: async () => (await api.snsList(tenant)).data,
    }),
  },
  invites: {
    list: (tenant: string) => ({
      queryKey: ['tenant-invite:list', tenant],
      queryFn: async () => (await api.tenantInviteList(tenant)).data,
    }),
  },
  workflows: {
    list: (tenant: string) => ({
      queryKey: ['workflow:list', tenant],
      queryFn: async () => (await api.workflowList(tenant)).data,
    }),
    get: (workflow: string) => ({
      queryKey: ['workflow:get', workflow],
      queryFn: async () => (await api.workflowGet(workflow)).data,
    }),
    getMetrics: (workflow: string) => ({
      queryKey: ['workflow:get:metrics', workflow],
      queryFn: async () => (await api.workflowGetMetrics(workflow)).data,
    }),
    getVersion: (workflow: string, version?: string) => ({
      queryKey: ['workflow-version:get', workflow, version],
      queryFn: async () =>
        (
          await api.workflowVersionGet(workflow, {
            version: version,
          })
        ).data,
    }),
    getDefinition: (workflow: string, version?: string) => ({
      queryKey: ['workflow-version:get:definition', workflow, version],
      queryFn: async () =>
        (
          await api.workflowVersionGetDefinition(workflow, {
            version: version,
          })
        ).data,
    }),
  },
  workflowRuns: {
    list: (tenant: string, query: ListWorkflowRunsQuery) => ({
      queryKey: ['workflow-run:list', tenant, query],
      queryFn: async () => (await api.workflowRunList(tenant, query)).data,
    }),
    get: (tenant: string, workflowRun: string) => ({
      queryKey: ['workflow-run:get', tenant, workflowRun],
      queryFn: async () => (await api.workflowRunGet(tenant, workflowRun)).data,
    }),
    metrics: (tenant: string, query: WorkflowRunMetrics) => ({
      queryKey: ['workflow-run:metrics', tenant, query],
      queryFn: async () =>
        (await api.workflowRunGetMetrics(tenant, query)).data,
    }),
    listPullRequests: (
      tenant: string,
      workflowRun: string,
      query: {
        state?: PullRequestState;
      },
    ) => ({
      queryKey: [
        'workflow-run:list:pull-requests',
        tenant,
        workflowRun,
        query.state,
      ],
      queryFn: async () =>
        (await api.workflowRunListPullRequests(tenant, workflowRun, query))
          .data,
    }),
  },
  stepRuns: {
    get: (tenant: string, stepRun: string) => ({
      queryKey: ['step-run:get', tenant, stepRun],
      queryFn: async () => (await api.stepRunGet(tenant, stepRun)).data,
    }),
    getDiff: (stepRun: string) => ({
      queryKey: ['step-run:get:diff', stepRun],
      queryFn: async () => (await api.stepRunGetDiff(stepRun)).data,
    }),
    getLogs: (stepRun: string, query: ListLogLineQuery) => ({
      queryKey: ['log-lines:list', stepRun],
      queryFn: async () => (await api.logLineList(stepRun, query)).data,
    }),
    getSchema: (tenant: string, stepRun: string) => ({
      queryKey: ['step-run:get:schema', stepRun],
      queryFn: async () => (await api.stepRunGetSchema(tenant, stepRun)).data,
    }),
  },
  events: {
    list: (tenant: string, query: ListEventQuery) => ({
      queryKey: ['event:list', tenant, query],
      queryFn: async () => (await api.eventList(tenant, query)).data,
    }),
    listKeys: (tenant: string) => ({
      queryKey: ['event-keys:list', tenant],
      queryFn: async () => (await api.eventKeyList(tenant)).data,
    }),
    getData: (event: string) => ({
      queryKey: ['event-data:get', event],
      queryFn: async () => (await api.eventDataGet(event)).data,
    }),
  },
  workers: {
    list: (tenant: string) => ({
      queryKey: ['worker:list', tenant],
      queryFn: async () => (await api.workerList(tenant)).data,
    }),
    get: (worker: string) => ({
      queryKey: ['worker:get', worker],
      queryFn: async () => (await api.workerGet(worker)).data,
    }),
  },
  github: {
    listInstallations: {
      queryKey: ['github-app:list:installations'],
      queryFn: async () => (await api.githubAppListInstallations()).data,
    },
    listRepos: (installation?: string) => ({
      queryKey: ['github-app:list:repos', installation],
      queryFn: async () => {
        invariant(installation, 'Installation must be set');
        const res = (await api.githubAppListRepos(installation)).data;
        return res;
      },
      enabled: !!installation,
    }),
    listBranches: (
      installation?: string,
      repoOwner?: string,
      repoName?: string,
    ) => ({
      queryKey: ['github-app:list:branches', installation, repoOwner, repoName],
      queryFn: async () => {
        invariant(installation, 'Installation must be set');
        invariant(repoOwner, 'Repo owner must be set');
        invariant(repoName, 'Repo name must be set');
        const res = (
          await api.githubAppListBranches(installation, repoOwner, repoName)
        ).data;
        return res;
      },
      enabled: !!installation && !!repoOwner && !!repoName,
    }),
  },
});
