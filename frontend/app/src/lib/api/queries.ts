import { createQueryKeyStore } from '@lukemorales/query-key-factory';

import api from './api';

type ListEventQuery = Parameters<typeof api.eventList>[1];
type ListWorkflowRunsQuery = Parameters<typeof api.workflowRunList>[1];

export const queries = createQueryKeyStore({
  user: {
    current: {
      queryKey: ['user:get'],
      queryFn: async () => (await api.userGetCurrent()).data,
    },
  },
  workflows: {
    list: (tenant: string) => ({
      queryKey: ['workflow:list', tenant],
      queryFn: async () => (await api.workflowList(tenant)).data,
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
});
