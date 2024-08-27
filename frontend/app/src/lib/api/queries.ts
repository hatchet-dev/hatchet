import { createQueryKeyStore } from '@lukemorales/query-key-factory';

import api, { cloudApi } from './api';
import invariant from 'tiny-invariant';
import { WebhookWorkerCreateRequest } from '.';

type ListEventQuery = Parameters<typeof api.eventList>[1];
type ListLogLineQuery = Parameters<typeof api.logLineList>[1];
type ListWorkflowRunsQuery = Parameters<typeof api.workflowRunList>[1];
export type ListCloudLogsQuery = Parameters<typeof cloudApi.logList>[1];
export type GetCloudMetricsQuery = Parameters<typeof cloudApi.metricsCpuGet>[1];
type WorkflowRunMetrics = Parameters<typeof api.workflowRunGetMetrics>[1];

export const queries = createQueryKeyStore({
  cloud: {
    billing: (tenant: string) => ({
      queryKey: ['billing-state:get', tenant],
      queryFn: async () => (await cloudApi.tenantBillingStateGet(tenant)).data,
    }),
    getManagedWorker: (worker: string) => ({
      queryKey: ['managed-worker:get', worker],
      queryFn: async () => (await cloudApi.managedWorkerGet(worker)).data,
    }),
    listManagedWorkers: (tenant: string) => ({
      queryKey: ['managed-worker:list', tenant],
      queryFn: async () => (await cloudApi.managedWorkerList(tenant)).data,
    }),
    listManagedWorkerInstances: (managedWorkerId: string) => ({
      queryKey: ['managed-worker:list:instances', managedWorkerId],
      queryFn: async () =>
        (await cloudApi.managedWorkerInstancesList(managedWorkerId)).data,
    }),
    getManagedWorkerLogs: (
      managedWorkerId: string,
      query: ListCloudLogsQuery,
    ) => ({
      queryKey: ['managed-worker:get:logs', managedWorkerId, query],
      queryFn: async () =>
        (await cloudApi.logList(managedWorkerId, query)).data,
    }),
    getBuild: (buildId: string) => ({
      queryKey: ['build:get', buildId],
      queryFn: async () => (await cloudApi.buildGet(buildId)).data,
    }),
    getBuildLogs: (buildId: string) => ({
      queryKey: ['build-logs:list', buildId],
      queryFn: async () => (await cloudApi.buildLogsList(buildId)).data,
    }),
    getManagedWorkerCpuMetrics: (
      managedWorkerId: string,
      query: GetCloudMetricsQuery,
    ) => ({
      queryKey: ['managed-worker:get:cpu-metrics', managedWorkerId, query],
      queryFn: async () =>
        (await cloudApi.metricsCpuGet(managedWorkerId, query)).data,
    }),
    getManagedWorkerMemoryMetrics: (
      managedWorkerId: string,
      query: GetCloudMetricsQuery,
    ) => ({
      queryKey: ['managed-worker:get:memory-metrics', managedWorkerId],
      queryFn: async () =>
        (await cloudApi.metricsMemoryGet(managedWorkerId, query)).data,
    }),
    getManagedWorkerDiskMetrics: (
      managedWorkerId: string,
      query: GetCloudMetricsQuery,
    ) => ({
      queryKey: ['managed-worker:get:disk-metrics', managedWorkerId],
      queryFn: async () =>
        (await cloudApi.metricsDiskGet(managedWorkerId, query)).data,
    }),
    listManagedWorkerEvents: (managedWorkerId: string) => ({
      queryKey: ['managed-worker:get:events', managedWorkerId],
      queryFn: async () =>
        (await cloudApi.managedWorkerEventsList(managedWorkerId)).data,
    }),
  },
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
  alertingSettings: {
    get: (tenant: string) => ({
      queryKey: ['tenant-alerting-settings:get', tenant],
      queryFn: async () => (await api.tenantAlertingSettingsGet(tenant)).data,
    }),
  },
  tenantResourcePolicy: {
    get: (tenant: string) => ({
      queryKey: ['tenant-resource-policy:get', tenant],
      queryFn: async () => (await api.tenantResourcePolicyGet(tenant)).data,
    }),
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
  emailGroups: {
    list: (tenant: string) => ({
      queryKey: ['email-group:list', tenant],
      queryFn: async () => (await api.alertEmailGroupList(tenant)).data,
    }),
  },
  slackWebhooks: {
    list: (tenant: string) => ({
      queryKey: ['slack-webhook:list', tenant],
      queryFn: async () => (await api.slackWebhookList(tenant)).data,
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
    getInput: (tenant: string, workflowRun: string) => ({
      queryKey: ['workflow-run:get:input', tenant, workflowRun],
      queryFn: async () => (await api.workflowRunGetInput(workflowRun)).data,
    }),
    metrics: (tenant: string, query: WorkflowRunMetrics) => ({
      queryKey: ['workflow-run:metrics', tenant, query],
      queryFn: async () =>
        (await api.workflowRunGetMetrics(tenant, query)).data,
    }),
  },
  stepRuns: {
    get: (tenant: string, stepRun: string) => ({
      queryKey: ['step-run:get', tenant, stepRun],
      queryFn: async () => (await api.stepRunGet(tenant, stepRun)).data,
    }),
    getLogs: (stepRun: string, query: ListLogLineQuery) => ({
      queryKey: ['log-lines:list', stepRun],
      queryFn: async () => (await api.logLineList(stepRun, query)).data,
    }),
    getSchema: (tenant: string, stepRun: string) => ({
      queryKey: ['step-run:get:schema', stepRun],
      queryFn: async () => (await api.stepRunGetSchema(tenant, stepRun)).data,
    }),
    listEvents: (stepRun: string) => ({
      queryKey: ['step-run:list:events', stepRun],
      queryFn: async () => (await api.stepRunListEvents(stepRun)).data,
    }),
    listArchives: (stepRun: string) => ({
      queryKey: ['step-run:list:archives', stepRun],
      queryFn: async () => (await api.stepRunListArchives(stepRun)).data,
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
    get: (
      worker: string,
      query: {
        recentFailed: boolean;
      },
    ) => ({
      queryKey: ['worker:get', worker, query.recentFailed],
      queryFn: async () => (await api.workerGet(worker, query)).data,
    }),
  },
  github: {
    listInstallations: {
      queryKey: ['github-app:list:installations'],
      queryFn: async () => (await cloudApi.githubAppListInstallations()).data,
    },
    listRepos: (installation?: string) => ({
      queryKey: ['github-app:list:repos', installation],
      queryFn: async () => {
        invariant(installation, 'Installation must be set');
        const res = (await cloudApi.githubAppListRepos(installation)).data;
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
          await cloudApi.githubAppListBranches(
            installation,
            repoOwner,
            repoName,
          )
        ).data;
        return res;
      },
      enabled: !!installation && !!repoOwner && !!repoName,
    }),
  },
  webhookWorkers: {
    list: (tenant: string) => ({
      queryKey: ['webhook-worker:list', tenant],
      queryFn: async () => (await api.webhookList(tenant)).data,
    }),
    create: (tenant: string, webhookWorker: WebhookWorkerCreateRequest) => ({
      queryKey: ['webhook-worker:create', tenant, webhookWorker],
      queryFn: async () =>
        (await api.webhookCreate(tenant, webhookWorker)).data,
    }),
    listRequests: (webhookWorkerId: string) => ({
      queryKey: ['webhook-worker:list:requests', webhookWorkerId],
      queryFn: async () =>
        (await api.webhookRequestsList(webhookWorkerId)).data,
    }),
  },
});
