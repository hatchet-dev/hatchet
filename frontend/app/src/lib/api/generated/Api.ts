/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

import {
  APIError,
  APIErrors,
  APIMeta,
  AcceptInviteRequest,
  BulkCreateEventRequest,
  CancelEventRequest,
  CreateAPITokenRequest,
  CreateAPITokenResponse,
  CreateCronWorkflowTriggerRequest,
  CreateEventRequest,
  CreateSNSIntegrationRequest,
  CreateTenantAlertEmailGroupRequest,
  CreateTenantInviteRequest,
  CreateTenantRequest,
  CronWorkflows,
  CronWorkflowsList,
  CronWorkflowsOrderByField,
  Event,
  EventData,
  EventKey,
  EventKeyList,
  EventList,
  EventOrderByDirection,
  EventOrderByField,
  EventSearch,
  Events,
  FeatureFlagEvaluationResult,
  FeatureFlagId,
  ListAPIMetaIntegration,
  ListAPITokensResponse,
  ListSNSIntegrations,
  ListSlackWebhooks,
  OtelSpanList,
  RateLimitList,
  RateLimitOrderByDirection,
  RateLimitOrderByField,
  RejectInviteRequest,
  ReplayEventRequest,
  ReplayWorkflowRunsRequest,
  ReplayWorkflowRunsResponse,
  RerunStepRunRequest,
  SNSIntegration,
  ScheduleWorkflowRunRequest,
  ScheduledRunStatus,
  ScheduledWorkflows,
  ScheduledWorkflowsBulkDeleteRequest,
  ScheduledWorkflowsBulkDeleteResponse,
  ScheduledWorkflowsBulkUpdateRequest,
  ScheduledWorkflowsBulkUpdateResponse,
  ScheduledWorkflowsList,
  ScheduledWorkflowsOrderByField,
  StepRun,
  StepRunArchiveList,
  StepRunEventList,
  TaskStats,
  Tenant,
  TenantAlertEmailGroup,
  TenantAlertEmailGroupList,
  TenantAlertingSettings,
  TenantInvite,
  TenantInviteList,
  TenantMember,
  TenantMemberList,
  TenantQueueMetrics,
  TenantResourcePolicy,
  TenantStepRunQueueMetrics,
  TriggerWorkflowRunRequest,
  UpdateCronWorkflowTriggerRequest,
  UpdateScheduledWorkflowRunRequest,
  UpdateTenantAlertEmailGroupRequest,
  UpdateTenantInviteRequest,
  UpdateTenantMemberRequest,
  UpdateTenantRequest,
  UpdateWorkerRequest,
  User,
  UserChangePasswordRequest,
  UserLoginRequest,
  UserRegisterRequest,
  UserTenantMembershipsList,
  V1BranchDurableTaskRequest,
  V1BranchDurableTaskResponse,
  V1CELDebugRequest,
  V1CELDebugResponse,
  V1CancelTaskRequest,
  V1CancelledTasks,
  V1CreateFilterRequest,
  V1CreateWebhookRequest,
  V1DagChildren,
  V1DurableEventLogList,
  V1Event,
  V1EventList,
  V1Filter,
  V1FilterList,
  V1LogLineLevel,
  V1LogLineList,
  V1LogLineOrderByDirection,
  V1LogsPointMetrics,
  V1ReplayTaskRequest,
  V1ReplayedTasks,
  V1RestoreTaskResponse,
  V1RunningFilter,
  V1TaskEventList,
  V1TaskPointMetrics,
  V1TaskRunMetrics,
  V1TaskStatus,
  V1TaskSummary,
  V1TaskSummaryList,
  V1TaskTimingList,
  V1TriggerWorkflowRunRequest,
  V1UpdateFilterRequest,
  V1UpdateWebhookRequest,
  V1Webhook,
  V1WebhookList,
  V1WebhookResponse,
  V1WebhookSourceName,
  V1WorkflowRunDetails,
  V1WorkflowRunDisplayNameList,
  V1WorkflowRunExternalIdList,
  WebhookWorkerCreateRequest,
  WebhookWorkerCreated,
  WebhookWorkerListResponse,
  WebhookWorkerRequestListResponse,
  Worker,
  WorkerList,
  Workflow,
  WorkflowID,
  WorkflowKindList,
  WorkflowList,
  WorkflowMetrics,
  WorkflowRun,
  WorkflowRunList,
  WorkflowRunOrderByDirection,
  WorkflowRunOrderByField,
  WorkflowRunShape,
  WorkflowRunStatus,
  WorkflowRunStatusList,
  WorkflowRunsCancelRequest,
  WorkflowRunsMetrics,
  WorkflowUpdateRequest,
  WorkflowVersion,
  WorkflowWorkersCount,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<
  SecurityDataType = unknown,
> extends HttpClient<SecurityDataType> {
  /**
   * @description Get a task by id
   *
   * @tags Task
   * @name V1TaskGet
   * @summary Get a task
   * @request GET:/api/v1/stable/tasks/{task}
   * @secure
   */
  v1TaskGet = Object.assign((
    task: string,
    query?: {
      /** The attempt number */
      attempt?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskSummary, APIErrors>({
      path: `/api/v1/stable/tasks/${task}`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "task"],
    }), { resources: new Set<string>(["tenant", "task"]) });
  /**
   * @description List events for a task
   *
   * @tags Task
   * @name V1TaskEventList
   * @summary List events for a task
   * @request GET:/api/v1/stable/tasks/{task}/task-events
   * @secure
   */
  v1TaskEventList = Object.assign((
    task: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskEventList, APIErrors>({
      path: `/api/v1/stable/tasks/${task}/task-events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "task"],
    }), { resources: new Set<string>(["tenant", "task"]) });
  /**
   * @description Lists log lines for a task
   *
   * @tags Log
   * @name V1LogLineList
   * @summary List log lines
   * @request GET:/api/v1/stable/tasks/{task}/logs
   * @secure
   */
  v1LogLineList = Object.assign((
    task: string,
    query?: {
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /**
       * The start time to get logs for
       * @format date-time
       */
      since?: string;
      /**
       * The end time to get logs for
       * @format date-time
       */
      until?: string;
      /** A full-text search query to filter for */
      search?: string;
      /** The log level(s) to include */
      levels?: V1LogLineLevel[];
      /** The direction to order by */
      order_by_direction?: V1LogLineOrderByDirection;
      /** The attempt number to filter for */
      attempt?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1LogLineList, APIErrors>({
      path: `/api/v1/stable/tasks/${task}/logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "task"],
    }), { resources: new Set<string>(["tenant", "task"]) });
  /**
   * @description Cancel tasks
   *
   * @tags Task
   * @name V1TaskCancel
   * @summary Cancel tasks
   * @request POST:/api/v1/stable/tenants/{tenant}/tasks/cancel
   * @secure
   */
  v1TaskCancel = Object.assign((
    tenant: string,
    data: V1CancelTaskRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1CancelledTasks, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/tasks/cancel`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists log lines for a tenant
   *
   * @tags Log
   * @name V1TenantLogLineList
   * @summary List log lines
   * @request GET:/api/v1/stable/tenants/{tenant}/logs
   * @secure
   */
  v1TenantLogLineList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /**
       * The start time to get logs for
       * @format date-time
       */
      since?: string;
      /**
       * The end time to get logs for
       * @format date-time
       */
      until?: string;
      /** A full-text search query to filter for */
      search?: string;
      /** The log level(s) to include */
      levels?: V1LogLineLevel[];
      /** The direction to order by */
      order_by_direction?: V1LogLineOrderByDirection;
      /** The attempt number to filter for */
      attempt?: number;
      /** The task external ID(s) to filter by */
      taskExternalIds?: string[];
      /** The workflow id(s) to filter for */
      workflow_ids?: string[];
      /** The step id(s) to filter for */
      step_ids?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1LogLineList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a minute by minute breakdown of log metrics for a tenant
   *
   * @tags Log
   * @name V1TenantLogLineGetPointMetrics
   * @summary Get log point metrics
   * @request GET:/api/v1/stable/tenants/{tenant}/log-point-metrics
   * @secure
   */
  v1TenantLogLineGetPointMetrics = Object.assign((
    tenant: string,
    query?: {
      /**
       * The start time to get logs for
       * @format date-time
       */
      since?: string;
      /**
       * The end time to get logs for
       * @format date-time
       */
      until?: string;
      /** A full-text search query to filter for */
      search?: string;
      /** The log level(s) to include */
      levels?: V1LogLineLevel[];
      /** The task external ID(s) to filter by */
      taskExternalIds?: string[];
      /** The workflow id(s) to filter for */
      workflow_ids?: string[];
      /** The step id(s) to filter for */
      step_ids?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1LogsPointMetrics, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/log-point-metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Replay tasks
   *
   * @tags Task
   * @name V1TaskReplay
   * @summary Replay tasks
   * @request POST:/api/v1/stable/tenants/{tenant}/tasks/replay
   * @secure
   */
  v1TaskReplay = Object.assign((
    tenant: string,
    data: V1ReplayTaskRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1ReplayedTasks, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/tasks/replay`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Restore an evicted durable task
   *
   * @tags Task
   * @name V1TaskRestore
   * @summary Restore a task
   * @request POST:/api/v1/stable/tasks/{task}/restore
   * @secure
   */
  v1TaskRestore = Object.assign((task: string, params: RequestParams = {}) =>
    this.request<V1RestoreTaskResponse, APIErrors>({
      path: `/api/v1/stable/tasks/${task}/restore`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "task"],
    }), { resources: new Set<string>(["tenant", "task"]) });
  /**
   * @description Lists all tasks that belong a specific list of dags
   *
   * @tags Task
   * @name V1DagListTasks
   * @summary List tasks
   * @request GET:/api/v1/stable/dags/tasks
   * @secure
   */
  v1DagListTasks = Object.assign((
    query: {
      /** The external id of the DAG */
      dag_ids: string[];
      /**
       * The tenant id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      tenant: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1DagChildren[], APIErrors>({
      path: `/api/v1/stable/dags/tasks`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Lists workflow runs for a tenant.
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunList
   * @summary List workflow runs
   * @request GET:/api/v1/stable/tenants/{tenant}/workflow-runs
   * @secure
   */
  v1WorkflowRunList = Object.assign((
    tenant: string,
    query: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** A list of statuses to filter by */
      statuses?: V1TaskStatus[];
      /**
       * The earliest date to filter by
       * @format date-time
       */
      since: string;
      /**
       * The latest date to filter by
       * @format date-time
       */
      until?: string;
      /** Additional metadata k-v pairs to filter by */
      additional_metadata?: string[];
      /** The workflow ids to find runs for */
      workflow_ids?: string[];
      /**
       * The worker id to filter by
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      worker_id?: string;
      /** Whether to include DAGs or only to include tasks */
      only_tasks: boolean;
      /**
       * The parent task external id to filter by
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parent_task_external_id?: string;
      /**
       * The external id of the event that triggered the workflow run
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      triggering_event_external_id?: string;
      /** A flag for whether or not to include the input and output payloads in the response. Defaults to `true` if unset. */
      include_payloads?: boolean;
      /** Filter within the RUNNING status bucket. ALL returns both on-worker and evicted tasks, ON_WORKER returns only tasks running on a worker, EVICTED returns only evicted tasks. Defaults to ALL. */
      running_filter?: V1RunningFilter;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskSummaryList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/workflow-runs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists displayable names of workflow runs for a tenant
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunDisplayNamesList
   * @summary List workflow runs
   * @request GET:/api/v1/stable/tenants/{tenant}/workflow-runs/display-names
   * @secure
   */
  v1WorkflowRunDisplayNamesList = Object.assign((
    tenant: string,
    query: {
      /** The external ids of the workflow runs to get display names for */
      external_ids: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1WorkflowRunDisplayNameList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/workflow-runs/display-names`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists external ids for workflow runs matching filters
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunExternalIdsList
   * @summary List workflow run external ids
   * @request GET:/api/v1/stable/tenants/{tenant}/workflow-runs/external-ids
   * @secure
   */
  v1WorkflowRunExternalIdsList = Object.assign((
    tenant: string,
    query: {
      /** A list of statuses to filter by */
      statuses?: V1TaskStatus[];
      /**
       * The earliest date to filter by
       * @format date-time
       */
      since: string;
      /**
       * The latest date to filter by
       * @format date-time
       */
      until?: string;
      /** Additional metadata k-v pairs to filter by */
      additional_metadata?: string[];
      /** The workflow ids to find runs for */
      workflow_ids?: string[];
      /** Filter within the RUNNING status bucket. ALL returns both on-worker and evicted tasks, ON_WORKER returns only tasks running on a worker, EVICTED returns only evicted tasks. Defaults to ALL. */
      running_filter?: V1RunningFilter;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1WorkflowRunExternalIdList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/workflow-runs/external-ids`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Trigger a new workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunCreate
   * @summary Create workflow run
   * @request POST:/api/v1/stable/tenants/{tenant}/workflow-runs/trigger
   * @secure
   */
  v1WorkflowRunCreate = Object.assign((
    tenant: string,
    data: V1TriggerWorkflowRunRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1WorkflowRunDetails, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/workflow-runs/trigger`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Branch a durable task from a specific node, creating a new branch and re-processing its matches.
   *
   * @tags Workflow Runs
   * @name V1DurableTaskBranch
   * @summary Branch durable task
   * @request POST:/api/v1/stable/tenants/{tenant}/durable-tasks/branch
   * @secure
   */
  v1DurableTaskBranch = Object.assign((
    tenant: string,
    data: V1BranchDurableTaskRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1BranchDurableTaskResponse, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/durable-tasks/branch`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists all event log entries for a durable task.
   *
   * @tags Durable Tasks
   * @name V1DurableTaskEventLogList
   * @summary List durable event log
   * @request GET:/api/v1/stable/durable-tasks/{durable-task}
   * @secure
   */
  v1DurableTaskEventLogList = Object.assign((
    durableTask: string,
    query?: {
      /**
       * The number of event log entries to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number of event log entries to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1DurableEventLogList, APIErrors>({
      path: `/api/v1/stable/durable-tasks/${durableTask}`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["durable-task"],
    }), { resources: new Set<string>(["durable-task"]) });
  /**
   * @description Get a workflow run and its metadata to display on the "detail" page
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunGet
   * @summary List tasks
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}
   * @secure
   */
  v1WorkflowRunGet = Object.assign((v1WorkflowRun: string, params: RequestParams = {}) =>
    this.request<V1WorkflowRunDetails, APIErrors>({
      path: `/api/v1/stable/workflow-runs/${v1WorkflowRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-workflow-run"],
    }), { resources: new Set<string>(["tenant", "v1-workflow-run"]) });
  /**
   * @description Get the status of a workflow run.
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunGetStatus
   * @summary Get workflow run status
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/status
   * @secure
   */
  v1WorkflowRunGetStatus = Object.assign((
    v1WorkflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<V1TaskStatus, APIErrors>({
      path: `/api/v1/stable/workflow-runs/${v1WorkflowRun}/status`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-workflow-run"],
    }), { resources: new Set<string>(["tenant", "v1-workflow-run"]) });
  /**
   * @description List all tasks for a workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunTaskEventsList
   * @summary List tasks
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/task-events
   * @secure
   */
  v1WorkflowRunTaskEventsList = Object.assign((
    v1WorkflowRun: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskEventList, APIErrors>({
      path: `/api/v1/stable/workflow-runs/${v1WorkflowRun}/task-events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-workflow-run"],
    }), { resources: new Set<string>(["tenant", "v1-workflow-run"]) });
  /**
   * @description Get OTel trace for a workflow run
   *
   * @tags Observability
   * @name V1ObservabilityGetTrace
   * @summary Get OTel trace
   * @request GET:/api/v1/stable/tenants/{tenant}/traces
   * @secure
   */
  v1ObservabilityGetTrace = Object.assign((
    tenant: string,
    query: {
      /**
       * The workflow run external id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      run_external_id: string;
      /**
       * The number of spans to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number of spans to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<OtelSpanList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/traces`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get the timings for a workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunGetTimings
   * @summary List timings for a workflow run
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/task-timings
   * @secure
   */
  v1WorkflowRunGetTimings = Object.assign((
    v1WorkflowRun: string,
    query?: {
      /**
       * The depth to retrieve children
       * @format int64
       */
      depth?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskTimingList, APIErrors>({
      path: `/api/v1/stable/workflow-runs/${v1WorkflowRun}/task-timings`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-workflow-run"],
    }), { resources: new Set<string>(["tenant", "v1-workflow-run"]) });
  /**
   * @description Get a summary of task run metrics for a tenant
   *
   * @tags Task
   * @name V1TaskListStatusMetrics
   * @summary Get task metrics
   * @request GET:/api/v1/stable/tenants/{tenant}/task-metrics
   * @secure
   */
  v1TaskListStatusMetrics = Object.assign((
    tenant: string,
    query: {
      /**
       * The start time to get metrics for
       * @format date-time
       */
      since: string;
      /**
       * The end time to get metrics for
       * @format date-time
       */
      until?: string;
      /** The workflow id to find runs for */
      workflow_ids?: string[];
      /**
       * The parent task's external id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parent_task_external_id?: string;
      /**
       * The id of the event that triggered the task
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      triggering_event_external_id?: string;
      /** Additional metadata k-v pairs to filter by */
      additional_metadata?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskRunMetrics, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/task-metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a minute by minute breakdown of task metrics for a tenant
   *
   * @tags Task
   * @name V1TaskGetPointMetrics
   * @summary Get task point metrics
   * @request GET:/api/v1/stable/tenants/{tenant}/task-point-metrics
   * @secure
   */
  v1TaskGetPointMetrics = Object.assign((
    tenant: string,
    query?: {
      /**
       * The time after the task was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdAfter?: string;
      /**
       * The time before the task was completed
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      finishedBefore?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1TaskPointMetrics, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/task-point-metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists all events for a tenant.
   *
   * @tags Event
   * @name V1EventList
   * @summary List events
   * @request GET:/api/v1/stable/tenants/{tenant}/events
   * @secure
   */
  v1EventList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** A list of keys to filter by */
      keys?: EventKey[];
      /**
       * Consider events that occurred after this time
       * @format date-time
       */
      since?: string;
      /**
       * Consider events that occurred before this time
       * @format date-time
       */
      until?: string;
      /** Filter to events that are associated with a specific workflow run */
      workflowIds?: string[];
      /** Filter to events that are associated with workflow runs matching a certain status */
      workflowRunStatuses?: V1TaskStatus[];
      /** Filter to specific events by their ids */
      eventIds?: string[];
      /** Filter by additional metadata on the events */
      additionalMetadata?: string[];
      /** The scopes to filter by */
      scopes?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1EventList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get an event by its id
   *
   * @tags Event
   * @name V1EventGet
   * @summary Get events
   * @request GET:/api/v1/stable/tenants/{tenant}/events/{v1-event}
   * @secure
   */
  v1EventGet = Object.assign((tenant: string, v1Event: string, params: RequestParams = {}) =>
    this.request<V1Event, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/events/${v1Event}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-event"],
    }), { resources: new Set<string>(["tenant", "v1-event"]) });
  /**
   * @description Lists all event keys for a tenant.
   *
   * @tags Event
   * @name V1EventKeyList
   * @summary List event keys
   * @request GET:/api/v1/stable/tenants/{tenant}/events/keys
   * @secure
   */
  v1EventKeyList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<EventKeyList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/events/keys`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists all filters for a tenant.
   *
   * @tags Filter
   * @name V1FilterList
   * @summary List filters
   * @request GET:/api/v1/stable/tenants/{tenant}/filters
   * @secure
   */
  v1FilterList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** The workflow ids to filter by */
      workflowIds?: string[];
      /** The scopes to subset candidate filters by */
      scopes?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1FilterList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/filters`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a new filter
   *
   * @tags Filter
   * @name V1FilterCreate
   * @summary Create a filter
   * @request POST:/api/v1/stable/tenants/{tenant}/filters
   * @secure
   */
  v1FilterCreate = Object.assign((
    tenant: string,
    data: V1CreateFilterRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1Filter, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/filters`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a filter by its id
   *
   * @tags Filter
   * @name V1FilterGet
   * @summary Get a filter
   * @request GET:/api/v1/stable/tenants/{tenant}/filters/{v1-filter}
   * @secure
   */
  v1FilterGet = Object.assign((
    tenant: string,
    v1Filter: string,
    params: RequestParams = {},
  ) =>
    this.request<V1Filter, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/filters/${v1Filter}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-filter"],
    }), { resources: new Set<string>(["tenant", "v1-filter"]) });
  /**
   * @description Delete a filter
   *
   * @tags Filter
   * @name V1FilterDelete
   * @request DELETE:/api/v1/stable/tenants/{tenant}/filters/{v1-filter}
   * @secure
   */
  v1FilterDelete = Object.assign((
    tenant: string,
    v1Filter: string,
    params: RequestParams = {},
  ) =>
    this.request<V1Filter, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/filters/${v1Filter}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-filter"],
    }), { resources: new Set<string>(["tenant", "v1-filter"]) });
  /**
   * @description Update a filter
   *
   * @tags Filter
   * @name V1FilterUpdate
   * @request PATCH:/api/v1/stable/tenants/{tenant}/filters/{v1-filter}
   * @secure
   */
  v1FilterUpdate = Object.assign((
    tenant: string,
    v1Filter: string,
    data: V1UpdateFilterRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1Filter, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/filters/${v1Filter}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-filter"],
    }), { resources: new Set<string>(["tenant", "v1-filter"]) });
  /**
   * @description Lists all webhook for a tenant.
   *
   * @tags Webhook
   * @name V1WebhookList
   * @summary List webhooks
   * @request GET:/api/v1/stable/tenants/{tenant}/webhooks
   * @secure
   */
  v1WebhookList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** The source names to filter by */
      sourceNames?: V1WebhookSourceName[];
      /** The webhook names to filter by */
      webhookNames?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1WebhookList, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a new webhook
   *
   * @tags Webhook
   * @name V1WebhookCreate
   * @summary Create a webhook
   * @request POST:/api/v1/stable/tenants/{tenant}/webhooks
   * @secure
   */
  v1WebhookCreate = Object.assign((
    tenant: string,
    data: V1CreateWebhookRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1Webhook, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a webhook by its name
   *
   * @tags Webhook
   * @name V1WebhookGet
   * @summary Get a webhook
   * @request GET:/api/v1/stable/tenants/{tenant}/webhooks/{v1-webhook}
   * @secure
   */
  v1WebhookGet = Object.assign((
    tenant: string,
    v1Webhook: string,
    params: RequestParams = {},
  ) =>
    this.request<V1Webhook, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks/${v1Webhook}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-webhook"],
    }), { resources: new Set<string>(["tenant", "v1-webhook"]) });
  /**
   * @description Delete a webhook
   *
   * @tags Webhook
   * @name V1WebhookDelete
   * @request DELETE:/api/v1/stable/tenants/{tenant}/webhooks/{v1-webhook}
   * @secure
   */
  v1WebhookDelete = Object.assign((
    tenant: string,
    v1Webhook: string,
    params: RequestParams = {},
  ) =>
    this.request<V1Webhook, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks/${v1Webhook}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-webhook"],
    }), { resources: new Set<string>(["tenant", "v1-webhook"]) });
  /**
   * @description Post an incoming webhook message
   *
   * @tags Webhook
   * @name V1WebhookReceive
   * @summary Post a webhook message
   * @request POST:/api/v1/stable/tenants/{tenant}/webhooks/{v1-webhook}
   */
  v1WebhookReceive = Object.assign((
    tenant: string,
    v1Webhook: string,
    data?: any,
    params: RequestParams = {},
  ) =>
    this.request<V1WebhookResponse, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks/${v1Webhook}`,
      method: "POST",
      body: data,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-webhook"],
    }), { resources: new Set<string>(["tenant", "v1-webhook"]) });
  /**
   * @description Update a webhook
   *
   * @tags Webhook
   * @name V1WebhookUpdate
   * @summary Update a webhook
   * @request PATCH:/api/v1/stable/tenants/{tenant}/webhooks/{v1-webhook}
   * @secure
   */
  v1WebhookUpdate = Object.assign((
    tenant: string,
    v1Webhook: string,
    data: V1UpdateWebhookRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1Webhook, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/webhooks/${v1Webhook}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "v1-webhook"],
    }), { resources: new Set<string>(["tenant", "v1-webhook"]) });
  /**
   * @description Evaluate a CEL expression against provided input data.
   *
   * @tags CEL
   * @name V1CelDebug
   * @summary Debug a CEL expression
   * @request POST:/api/v1/stable/tenants/{tenant}/cel/debug
   * @secure
   */
  v1CelDebug = Object.assign((
    tenant: string,
    data: V1CELDebugRequest,
    params: RequestParams = {},
  ) =>
    this.request<V1CELDebugResponse, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/cel/debug`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets the readiness status
   *
   * @tags Healthcheck
   * @name ReadinessGet
   * @summary Get readiness
   * @request GET:/api/ready
   */
  readinessGet = Object.assign((params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/ready`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Gets the liveness status
   *
   * @tags Healthcheck
   * @name LivenessGet
   * @summary Get liveness
   * @request GET:/api/live
   */
  livenessGet = Object.assign((params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/live`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Gets metadata for the Hatchet instance
   *
   * @tags Metadata
   * @name MetadataGet
   * @summary Get metadata
   * @request GET:/api/v1/meta
   */
  metadataGet = Object.assign((params: RequestParams = {}) =>
    this.request<APIMeta, APIErrors>({
      path: `/api/v1/meta`,
      method: "GET",
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Gets metadata for the Hatchet cloud instance
   *
   * @tags Metadata
   * @name CloudMetadataGet
   * @summary Get cloud metadata
   * @request GET:/api/v1/cloud/metadata
   */
  cloudMetadataGet = Object.assign((params: RequestParams = {}) =>
    this.request<APIErrors, APIErrors>({
      path: `/api/v1/cloud/metadata`,
      method: "GET",
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description List all integrations
   *
   * @tags Metadata
   * @name MetadataListIntegrations
   * @summary List integrations
   * @request GET:/api/v1/meta/integrations
   * @secure
   */
  metadataListIntegrations = Object.assign((params: RequestParams = {}) =>
    this.request<ListAPIMetaIntegration, APIErrors>({
      path: `/api/v1/meta/integrations`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Logs in a user.
   *
   * @tags User
   * @name UserUpdateLogin
   * @summary Login user
   * @request POST:/api/v1/users/login
   */
  userUpdateLogin = Object.assign((data: UserLoginRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/login`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGoogleOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/users/google/start
   */
  userUpdateGoogleOauthStart = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/google/start`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGoogleOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/google/callback
   */
  userUpdateGoogleOauthCallback = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/google/callback`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/users/github/start
   */
  userUpdateGithubOauthStart = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/start`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/github/callback
   */
  userUpdateGithubOauthCallback = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/callback`,
      method: "GET",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateSlackOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/tenants/{tenant}/slack/start
   * @secure
   */
  userUpdateSlackOauthStart = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/tenants/${tenant}/slack/start`,
      method: "GET",
      secure: true,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateSlackOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/slack/callback
   * @secure
   */
  userUpdateSlackOauthCallback = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/slack/callback`,
      method: "GET",
      secure: true,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description SNS event
   *
   * @tags Github
   * @name SnsUpdate
   * @summary Github app tenant webhook
   * @request POST:/api/v1/sns/{tenant}/{event}
   */
  snsUpdate = Object.assign((tenant: string, event: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/sns/${tenant}/${event}`,
      method: "POST",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description List SNS integrations
   *
   * @tags SNS
   * @name SnsList
   * @summary List SNS integrations
   * @request GET:/api/v1/tenants/{tenant}/sns
   * @secure
   */
  snsList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<ListSNSIntegrations, APIErrors>({
      path: `/api/v1/tenants/${tenant}/sns`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create SNS integration
   *
   * @tags SNS
   * @name SnsCreate
   * @summary Create SNS integration
   * @request POST:/api/v1/tenants/{tenant}/sns
   * @secure
   */
  snsCreate = Object.assign((
    tenant: string,
    data: CreateSNSIntegrationRequest,
    params: RequestParams = {},
  ) =>
    this.request<SNSIntegration, APIErrors>({
      path: `/api/v1/tenants/${tenant}/sns`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Creates a new tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupCreate
   * @summary Create tenant alert email group
   * @request POST:/api/v1/tenants/{tenant}/alerting-email-groups
   * @secure
   */
  alertEmailGroupCreate = Object.assign((
    tenant: string,
    data: CreateTenantAlertEmailGroupRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantAlertEmailGroup, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/alerting-email-groups`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets a list of tenant alert email groups
   *
   * @tags Tenant
   * @name AlertEmailGroupList
   * @summary List tenant alert email groups
   * @request GET:/api/v1/tenants/{tenant}/alerting-email-groups
   * @secure
   */
  alertEmailGroupList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantAlertEmailGroupList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/alerting-email-groups`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets the resource policy for a tenant
   *
   * @tags Tenant
   * @name TenantResourcePolicyGet
   * @summary Create tenant alert email group
   * @request GET:/api/v1/tenants/{tenant}/resource-policy
   * @secure
   */
  tenantResourcePolicyGet = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantResourcePolicy, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/resource-policy`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Updates a tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupUpdate
   * @summary Update tenant alert email group
   * @request PATCH:/api/v1/alerting-email-groups/{alert-email-group}
   * @secure
   */
  alertEmailGroupUpdate = Object.assign((
    alertEmailGroup: string,
    data: UpdateTenantAlertEmailGroupRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantAlertEmailGroup, APIErrors | APIError>({
      path: `/api/v1/alerting-email-groups/${alertEmailGroup}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "alert-email-group"],
    }), { resources: new Set<string>(["tenant", "alert-email-group"]) });
  /**
   * @description Deletes a tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupDelete
   * @summary Delete tenant alert email group
   * @request DELETE:/api/v1/alerting-email-groups/{alert-email-group}
   * @secure
   */
  alertEmailGroupDelete = Object.assign((
    alertEmailGroup: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/alerting-email-groups/${alertEmailGroup}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "alert-email-group"],
    }), { resources: new Set<string>(["tenant", "alert-email-group"]) });
  /**
   * @description Delete SNS integration
   *
   * @tags SNS
   * @name SnsDelete
   * @summary Delete SNS integration
   * @request DELETE:/api/v1/sns/{sns}
   * @secure
   */
  snsDelete = Object.assign((sns: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/sns/${sns}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "sns"],
    }), { resources: new Set<string>(["tenant", "sns"]) });
  /**
   * @description List Slack webhooks
   *
   * @tags Slack
   * @name SlackWebhookList
   * @summary List Slack integrations
   * @request GET:/api/v1/tenants/{tenant}/slack
   * @secure
   */
  slackWebhookList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<ListSlackWebhooks, APIErrors>({
      path: `/api/v1/tenants/${tenant}/slack`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Delete Slack webhook
   *
   * @tags Slack
   * @name SlackWebhookDelete
   * @summary Delete Slack webhook
   * @request DELETE:/api/v1/slack/{slack}
   * @secure
   */
  slackWebhookDelete = Object.assign((slack: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/slack/${slack}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "slack"],
    }), { resources: new Set<string>(["tenant", "slack"]) });
  /**
   * @description Gets the current user
   *
   * @tags User
   * @name UserGetCurrent
   * @summary Get current user
   * @request GET:/api/v1/users/current
   * @secure
   */
  userGetCurrent = Object.assign((params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/current`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Update a user password.
   *
   * @tags User
   * @name UserUpdatePassword
   * @summary Change user password
   * @request POST:/api/v1/users/password
   * @secure
   */
  userUpdatePassword = Object.assign((
    data: UserChangePasswordRequest,
    params: RequestParams = {},
  ) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/password`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Registers a user.
   *
   * @tags User
   * @name UserCreate
   * @summary Register user
   * @request POST:/api/v1/users/register
   */
  userCreate = Object.assign((data: UserRegisterRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/register`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Logs out a user.
   *
   * @tags User
   * @name UserUpdateLogout
   * @summary Logout user
   * @request POST:/api/v1/users/logout
   * @secure
   */
  userUpdateLogout = Object.assign((params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/logout`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Lists all tenant memberships for the current user
   *
   * @tags User
   * @name TenantMembershipsList
   * @summary List tenant memberships
   * @request GET:/api/v1/users/memberships
   * @secure
   */
  tenantMembershipsList = Object.assign((params: RequestParams = {}) =>
    this.request<UserTenantMembershipsList, APIErrors>({
      path: `/api/v1/users/memberships`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Lists all tenant invites for the current user
   *
   * @tags Tenant
   * @name UserListTenantInvites
   * @summary List tenant invites
   * @request GET:/api/v1/users/invites
   * @secure
   */
  userListTenantInvites = Object.assign((params: RequestParams = {}) =>
    this.request<TenantInviteList, APIErrors>({
      path: `/api/v1/users/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Accepts a tenant invite
   *
   * @tags Tenant
   * @name TenantInviteAccept
   * @summary Accept tenant invite
   * @request POST:/api/v1/users/invites/accept
   * @secure
   */
  tenantInviteAccept = Object.assign((
    data: AcceptInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/users/invites/accept`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Rejects a tenant invite
   *
   * @tags Tenant
   * @name TenantInviteReject
   * @summary Reject tenant invite
   * @request POST:/api/v1/users/invites/reject
   * @secure
   */
  tenantInviteReject = Object.assign((
    data: RejectInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/users/invites/reject`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Creates a new tenant
   *
   * @tags Tenant
   * @name TenantCreate
   * @summary Create tenant
   * @request POST:/api/v1/tenants
   * @secure
   */
  tenantCreate = Object.assign((data: CreateTenantRequest, params: RequestParams = {}) =>
    this.request<Tenant, APIErrors | APIError>({
      path: `/api/v1/tenants`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Update an existing tenant
   *
   * @tags Tenant
   * @name TenantUpdate
   * @summary Update tenant
   * @request PATCH:/api/v1/tenants/{tenant}
   * @secure
   */
  tenantUpdate = Object.assign((
    tenant: string,
    data: UpdateTenantRequest,
    params: RequestParams = {},
  ) =>
    this.request<Tenant, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get the details of a tenant
   *
   * @tags Tenant
   * @name TenantGet
   * @summary Get tenant
   * @request GET:/api/v1/tenants/{tenant}
   * @secure
   */
  tenantGet = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<Tenant, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets the alerting settings for a tenant
   *
   * @tags Tenant
   * @name TenantAlertingSettingsGet
   * @summary Get tenant alerting settings
   * @request GET:/api/v1/tenants/{tenant}/alerting/settings
   * @secure
   */
  tenantAlertingSettingsGet = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantAlertingSettings, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/alerting/settings`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Creates a new tenant invite
   *
   * @tags Tenant
   * @name TenantInviteCreate
   * @summary Create tenant invite
   * @request POST:/api/v1/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteCreate = Object.assign((
    tenant: string,
    data: CreateTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantInvite, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/invites`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets a list of tenant invites
   *
   * @tags Tenant
   * @name TenantInviteList
   * @summary List tenant invites
   * @request GET:/api/v1/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantInviteList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Updates a tenant invite
   *
   * @name TenantInviteUpdate
   * @summary Update invite
   * @request PATCH:/api/v1/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteUpdate = Object.assign((
    tenant: string,
    tenantInvite: string,
    data: UpdateTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantInvite, APIErrors>({
      path: `/api/v1/tenants/${tenant}/invites/${tenantInvite}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "tenant-invite"],
    }), { resources: new Set<string>(["tenant", "tenant-invite"]) });
  /**
   * @description Deletes a tenant invite
   *
   * @name TenantInviteDelete
   * @summary Delete invite
   * @request DELETE:/api/v1/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteDelete = Object.assign((
    tenant: string,
    tenantInvite: string,
    params: RequestParams = {},
  ) =>
    this.request<TenantInvite, APIErrors>({
      path: `/api/v1/tenants/${tenant}/invites/${tenantInvite}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "tenant-invite"],
    }), { resources: new Set<string>(["tenant", "tenant-invite"]) });
  /**
   * @description Create an API token for a tenant
   *
   * @tags API Token
   * @name ApiTokenCreate
   * @summary Create API Token
   * @request POST:/api/v1/tenants/{tenant}/api-tokens
   * @secure
   */
  apiTokenCreate = Object.assign((
    tenant: string,
    data: CreateAPITokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<CreateAPITokenResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/api-tokens`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description List API tokens for a tenant
   *
   * @tags API Token
   * @name ApiTokenList
   * @summary List API Tokens
   * @request GET:/api/v1/tenants/{tenant}/api-tokens
   * @secure
   */
  apiTokenList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<ListAPITokensResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/api-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Revoke an API token for a tenant
   *
   * @tags API Token
   * @name ApiTokenUpdateRevoke
   * @summary Revoke API Token
   * @request POST:/api/v1/api-tokens/{api-token}
   * @secure
   */
  apiTokenUpdateRevoke = Object.assign((apiToken: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/api-tokens/${apiToken}`,
      method: "POST",
      secure: true,
      ...params,
      xResources: ["tenant", "api-token"],
    }), { resources: new Set<string>(["tenant", "api-token"]) });
  /**
   * @description Get the queue metrics for the tenant
   *
   * @tags Workflow
   * @name TenantGetQueueMetrics
   * @summary Get workflow metrics
   * @request GET:/api/v1/tenants/{tenant}/queue-metrics
   * @secure
   */
  tenantGetQueueMetrics = Object.assign((
    tenant: string,
    query?: {
      /** A list of workflow IDs to filter by */
      workflows?: WorkflowID[];
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<TenantQueueMetrics, APIErrors>({
      path: `/api/v1/tenants/${tenant}/queue-metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get the queue metrics for the tenant
   *
   * @tags Tenant
   * @name TenantGetStepRunQueueMetrics
   * @summary Get step run metrics
   * @request GET:/api/v1/tenants/{tenant}/step-run-queue-metrics
   * @secure
   */
  tenantGetStepRunQueueMetrics = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantStepRunQueueMetrics, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-run-queue-metrics`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists all events for a tenant.
   *
   * @tags Event
   * @name EventList
   * @summary List events
   * @request GET:/api/v1/tenants/{tenant}/events
   * @secure
   */
  eventList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** A list of keys to filter by */
      keys?: EventKey[];
      /** A list of workflow IDs to filter by */
      workflows?: WorkflowID[];
      /** A list of workflow run statuses to filter by */
      statuses?: WorkflowRunStatusList;
      /** The search query to filter for */
      search?: EventSearch;
      /** What to order by */
      orderByField?: EventOrderByField;
      /** The order direction */
      orderByDirection?: EventOrderByDirection;
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
      /** A list of event ids to filter by */
      eventIds?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Creates a new event.
   *
   * @tags Event
   * @name EventCreate
   * @summary Create event
   * @request POST:/api/v1/tenants/{tenant}/events
   * @secure
   */
  eventCreate = Object.assign((
    tenant: string,
    data: CreateEventRequest,
    params: RequestParams = {},
  ) =>
    this.request<Event, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Bulk creates new events.
   *
   * @tags Event
   * @name EventCreateBulk
   * @summary Bulk Create events
   * @request POST:/api/v1/tenants/{tenant}/events/bulk
   * @secure
   */
  eventCreateBulk = Object.assign((
    tenant: string,
    data: BulkCreateEventRequest,
    params: RequestParams = {},
  ) =>
    this.request<Events, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/bulk`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Replays a list of events.
   *
   * @tags Event
   * @name EventUpdateReplay
   * @summary Replay events
   * @request POST:/api/v1/tenants/{tenant}/events/replay
   * @secure
   */
  eventUpdateReplay = Object.assign((
    tenant: string,
    data: ReplayEventRequest,
    params: RequestParams = {},
  ) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/replay`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Cancels all runs for a list of events.
   *
   * @tags Event
   * @name EventUpdateCancel
   * @summary Replay events
   * @request POST:/api/v1/tenants/{tenant}/events/cancel
   * @secure
   */
  eventUpdateCancel = Object.assign((
    tenant: string,
    data: CancelEventRequest,
    params: RequestParams = {},
  ) =>
    this.request<
      {
        workflowRunIds?: string[];
      },
      APIErrors
    >({
      path: `/api/v1/tenants/${tenant}/events/cancel`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists all rate limits for a tenant.
   *
   * @tags Rate Limits
   * @name RateLimitList
   * @summary List rate limits
   * @request GET:/api/v1/tenants/{tenant}/rate-limits
   * @secure
   */
  rateLimitList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** The search query to filter for */
      search?: string;
      /** What to order by */
      orderByField?: RateLimitOrderByField;
      /** The order direction */
      orderByDirection?: RateLimitOrderByDirection;
    },
    params: RequestParams = {},
  ) =>
    this.request<RateLimitList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/rate-limits`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Delete a rate limit for a tenant.
   *
   * @tags Rate Limits
   * @name RateLimitDelete
   * @summary Delete rate limit
   * @request DELETE:/api/v1/tenants/{tenant}/rate-limits
   * @secure
   */
  rateLimitDelete = Object.assign((
    tenant: string,
    query: {
      /** The limit key */
      key: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/tenants/${tenant}/rate-limits`,
      method: "DELETE",
      query: query,
      secure: true,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Gets a list of tenant members
   *
   * @tags Tenant
   * @name TenantMemberList
   * @summary List tenant members
   * @request GET:/api/v1/tenants/{tenant}/members
   * @secure
   */
  tenantMemberList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<TenantMemberList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/members`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Update a tenant member
   *
   * @tags Tenant
   * @name TenantMemberUpdate
   * @summary Update a tenant member
   * @request PATCH:/api/v1/tenants/{tenant}/members/{member}
   * @secure
   */
  tenantMemberUpdate = Object.assign((
    tenant: string,
    member: string,
    data: UpdateTenantMemberRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantMember, APIErrors>({
      path: `/api/v1/tenants/${tenant}/members/${member}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "member"],
    }), { resources: new Set<string>(["tenant", "member"]) });
  /**
   * @description Delete a member from a tenant
   *
   * @tags Tenant
   * @name TenantMemberDelete
   * @summary Delete a tenant member
   * @request DELETE:/api/v1/tenants/{tenant}/members/{member}
   * @secure
   */
  tenantMemberDelete = Object.assign((
    tenant: string,
    member: string,
    params: RequestParams = {},
  ) =>
    this.request<TenantMember, APIErrors>({
      path: `/api/v1/tenants/${tenant}/members/${member}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "member"],
    }), { resources: new Set<string>(["tenant", "member"]) });
  /**
   * @description Get an event.
   *
   * @tags Event
   * @name EventGet
   * @summary Get event data
   * @request GET:/api/v1/events/{event}
   * @secure
   */
  eventGet = Object.assign((event: string, params: RequestParams = {}) =>
    this.request<Event, APIErrors>({
      path: `/api/v1/events/${event}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "event"],
    }), { resources: new Set<string>(["tenant", "event"]) });
  /**
   * @description Get the data for an event.
   *
   * @tags Event
   * @name EventDataGet
   * @summary Get event data
   * @request GET:/api/v1/events/{event}/data
   * @secure
   */
  eventDataGet = Object.assign((event: string, params: RequestParams = {}) =>
    this.request<EventData, APIErrors>({
      path: `/api/v1/events/${event}/data`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "event"],
    }), { resources: new Set<string>(["tenant", "event"]) });
  /**
   * @description Get the data for an event.
   *
   * @tags Event
   * @name EventDataGetWithTenant
   * @summary Get event data
   * @request GET:/api/v1/tenants/{tenant}/events/{event-with-tenant}/data
   * @secure
   */
  eventDataGetWithTenant = Object.assign((
    eventWithTenant: string,
    tenant: string,
    params: RequestParams = {},
  ) =>
    this.request<EventData, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/${eventWithTenant}/data`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "event-with-tenant"],
    }), { resources: new Set<string>(["tenant", "event-with-tenant"]) });
  /**
   * @description Lists all event keys for a tenant.
   *
   * @tags Event
   * @name EventKeyList
   * @summary List event keys
   * @request GET:/api/v1/tenants/{tenant}/events/keys
   * @secure
   */
  eventKeyList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<EventKeyList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/keys`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get all workflows for a tenant
   *
   * @tags Workflow
   * @name WorkflowList
   * @summary Get workflows
   * @request GET:/api/v1/tenants/{tenant}/workflows
   * @secure
   */
  workflowList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int
       * @default 0
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int
       * @default 50
       */
      limit?: number;
      /** Search by name */
      name?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Schedule a new workflow run for a tenant
   *
   * @tags Workflow Run
   * @name ScheduledWorkflowRunCreate
   * @summary Trigger workflow run
   * @request POST:/api/v1/tenants/{tenant}/workflows/{workflow}/scheduled
   * @secure
   */
  scheduledWorkflowRunCreate = Object.assign((
    tenant: string,
    workflow: string,
    data: ScheduleWorkflowRunRequest,
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflows, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/${workflow}/scheduled`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get all scheduled workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledList
   * @summary Get scheduled workflow runs
   * @request GET:/api/v1/tenants/{tenant}/workflows/scheduled
   * @secure
   */
  workflowScheduledList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** The order by field */
      orderByField?: ScheduledWorkflowsOrderByField;
      /** The order by direction */
      orderByDirection?: WorkflowRunOrderByDirection;
      /**
       * The workflow id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      workflowId?: string;
      /**
       * The parent workflow run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentWorkflowRunId?: string;
      /**
       * The parent step run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentStepRunId?: string;
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
      /** A list of scheduled run statuses to filter by */
      statuses?: ScheduledRunStatus[];
    },
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflowsList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a scheduled workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledGet
   * @summary Get scheduled workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}
   * @secure
   */
  workflowScheduledGet = Object.assign((
    tenant: string,
    scheduledWorkflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflows, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/${scheduledWorkflowRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "scheduled-workflow-run"],
    }), { resources: new Set<string>(["tenant", "scheduled-workflow-run"]) });
  /**
   * @description Delete a scheduled workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledDelete
   * @summary Delete scheduled workflow run
   * @request DELETE:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}
   * @secure
   */
  workflowScheduledDelete = Object.assign((
    tenant: string,
    scheduledWorkflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/${scheduledWorkflowRun}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "scheduled-workflow-run"],
    }), { resources: new Set<string>(["tenant", "scheduled-workflow-run"]) });
  /**
   * @description Update (reschedule) a scheduled workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledUpdate
   * @summary Update scheduled workflow run
   * @request PATCH:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}
   * @secure
   */
  workflowScheduledUpdate = Object.assign((
    tenant: string,
    scheduledWorkflowRun: string,
    data: UpdateScheduledWorkflowRunRequest,
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflows, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/${scheduledWorkflowRun}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "scheduled-workflow-run"],
    }), { resources: new Set<string>(["tenant", "scheduled-workflow-run"]) });
  /**
   * @description Bulk delete scheduled workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledBulkDelete
   * @summary Bulk delete scheduled workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflows/scheduled/bulk-delete
   * @secure
   */
  workflowScheduledBulkDelete = Object.assign((
    tenant: string,
    data: ScheduledWorkflowsBulkDeleteRequest,
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflowsBulkDeleteResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/bulk-delete`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Bulk update (reschedule) scheduled workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledBulkUpdate
   * @summary Bulk update scheduled workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflows/scheduled/bulk-update
   * @secure
   */
  workflowScheduledBulkUpdate = Object.assign((
    tenant: string,
    data: ScheduledWorkflowsBulkUpdateRequest,
    params: RequestParams = {},
  ) =>
    this.request<ScheduledWorkflowsBulkUpdateResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/bulk-update`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a new cron job workflow trigger for a tenant
   *
   * @tags Workflow Run
   * @name CronWorkflowTriggerCreate
   * @summary Create cron job workflow trigger
   * @request POST:/api/v1/tenants/{tenant}/workflows/{workflow}/crons
   * @secure
   */
  cronWorkflowTriggerCreate = Object.assign((
    tenant: string,
    workflow: string,
    data: CreateCronWorkflowTriggerRequest,
    params: RequestParams = {},
  ) =>
    this.request<CronWorkflows, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/${workflow}/crons`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get all cron job workflow triggers for a tenant
   *
   * @tags Workflow
   * @name CronWorkflowList
   * @summary Get cron job workflows
   * @request GET:/api/v1/tenants/{tenant}/workflows/crons
   * @secure
   */
  cronWorkflowList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /**
       * The workflow id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      workflowId?: string;
      /** The workflow name to get runs for. */
      workflowName?: string;
      /** The cron name to get runs for. */
      cronName?: string;
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
      /** The order by field */
      orderByField?: CronWorkflowsOrderByField;
      /** The order by direction */
      orderByDirection?: WorkflowRunOrderByDirection;
    },
    params: RequestParams = {},
  ) =>
    this.request<CronWorkflowsList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/crons`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a cron job workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowCronGet
   * @summary Get cron job workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}
   * @secure
   */
  workflowCronGet = Object.assign((
    tenant: string,
    cronWorkflow: string,
    params: RequestParams = {},
  ) =>
    this.request<CronWorkflows, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/crons/${cronWorkflow}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "cron-workflow"],
    }), { resources: new Set<string>(["tenant", "cron-workflow"]) });
  /**
   * @description Delete a cron job workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowCronDelete
   * @summary Delete cron job workflow run
   * @request DELETE:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}
   * @secure
   */
  workflowCronDelete = Object.assign((
    tenant: string,
    cronWorkflow: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/workflows/crons/${cronWorkflow}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "cron-workflow"],
    }), { resources: new Set<string>(["tenant", "cron-workflow"]) });
  /**
   * @description Update a cron workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowCronUpdate
   * @summary Update cron job workflow run
   * @request PATCH:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}
   * @secure
   */
  workflowCronUpdate = Object.assign((
    tenant: string,
    cronWorkflow: string,
    data: UpdateCronWorkflowTriggerRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/workflows/crons/${cronWorkflow}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["tenant", "cron-workflow"],
    }), { resources: new Set<string>(["tenant", "cron-workflow"]) });
  /**
   * @description Cancel a batch of workflow runs
   *
   * @tags Workflow Run
   * @name WorkflowRunCancel
   * @summary Cancel workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflows/cancel
   * @secure
   */
  workflowRunCancel = Object.assign((
    tenant: string,
    data: WorkflowRunsCancelRequest,
    params: RequestParams = {},
  ) =>
    this.request<
      {
        workflowRunIds?: string[];
      },
      APIErrors
    >({
      path: `/api/v1/tenants/${tenant}/workflows/cancel`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowGet
   * @summary Get workflow
   * @request GET:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowGet = Object.assign((workflow: string, params: RequestParams = {}) =>
    this.request<Workflow, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Delete a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowDelete
   * @summary Delete workflow
   * @request DELETE:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowDelete = Object.assign((workflow: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Update a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowUpdate
   * @summary Update workflow
   * @request PATCH:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowUpdate = Object.assign((
    workflow: string,
    data: WorkflowUpdateRequest,
    params: RequestParams = {},
  ) =>
    this.request<Workflow, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Get a workflow version for a tenant
   *
   * @tags Workflow
   * @name WorkflowVersionGet
   * @summary Get workflow version
   * @request GET:/api/v1/workflows/{workflow}/versions
   * @secure
   */
  workflowVersionGet = Object.assign((
    workflow: string,
    query?: {
      /**
       * The workflow version. If not supplied, the latest version is fetched.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      version?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowVersion, APIErrors>({
      path: `/api/v1/workflows/${workflow}/versions`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Trigger a new workflow run for a tenant
   *
   * @tags Workflow Run
   * @name WorkflowRunCreate
   * @summary Trigger workflow run
   * @request POST:/api/v1/workflows/{workflow}/trigger
   * @secure
   */
  workflowRunCreate = Object.assign((
    workflow: string,
    data: TriggerWorkflowRunRequest,
    query?: {
      /**
       * The workflow version. If not supplied, the latest version is fetched.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      version?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRun, APIErrors>({
      path: `/api/v1/workflows/${workflow}/trigger`,
      method: "POST",
      query: query,
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Get the metrics for a workflow version
   *
   * @tags Workflow
   * @name WorkflowGetMetrics
   * @summary Get workflow metrics
   * @request GET:/api/v1/workflows/{workflow}/metrics
   * @secure
   */
  workflowGetMetrics = Object.assign((
    workflow: string,
    query?: {
      /** A status of workflow run statuses to filter by */
      status?: WorkflowRunStatus;
      /** A group key to filter metrics by */
      groupKey?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowMetrics, APIErrors>({
      path: `/api/v1/workflows/${workflow}/metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description List events for a step run
   *
   * @tags Step Run
   * @name StepRunListEvents
   * @summary List events for step run
   * @request GET:/api/v1/step-runs/{step-run}/events
   * @secure
   */
  stepRunListEvents = Object.assign((
    stepRun: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<StepRunEventList, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description List events for all step runs for a workflow run
   *
   * @tags Step Run
   * @name WorkflowRunListStepRunEvents
   * @summary List events for all step runs for a workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/step-run-events
   * @secure
   */
  workflowRunListStepRunEvents = Object.assign((
    tenant: string,
    workflowRun: string,
    query?: {
      /**
       * Last ID of the last event
       * @format int32
       */
      lastId?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<StepRunEventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}/step-run-events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description List archives for a step run
   *
   * @tags Step Run
   * @name StepRunListArchives
   * @summary List archives for step run
   * @request GET:/api/v1/step-runs/{step-run}/archives
   * @secure
   */
  stepRunListArchives = Object.assign((
    stepRun: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
    },
    params: RequestParams = {},
  ) =>
    this.request<StepRunArchiveList, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/archives`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description Get a count of the workers available for workflow
   *
   * @tags Workflow
   * @name WorkflowGetWorkersCount
   * @summary Get workflow worker count
   * @request GET:/api/v1/tenants/{tenant}/workflows/{workflow}/worker-count
   * @secure
   */
  workflowGetWorkersCount = Object.assign((
    tenant: string,
    workflow: string,
    params: RequestParams = {},
  ) =>
    this.request<WorkflowWorkersCount, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/${workflow}/worker-count`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow"],
    }), { resources: new Set<string>(["tenant", "workflow"]) });
  /**
   * @description Get all workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunList
   * @summary Get workflow runs
   * @request GET:/api/v1/tenants/{tenant}/workflows/runs
   * @secure
   */
  workflowRunList = Object.assign((
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /**
       * The event id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      eventId?: string;
      /**
       * The workflow id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      workflowId?: string;
      /**
       * The parent workflow run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentWorkflowRunId?: string;
      /**
       * The parent step run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentStepRunId?: string;
      /** A list of workflow run statuses to filter by */
      statuses?: WorkflowRunStatusList;
      /** A list of workflow kinds to filter by */
      kinds?: WorkflowKindList;
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
      /**
       * The time after the workflow run was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdAfter?: string;
      /**
       * The time before the workflow run was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdBefore?: string;
      /**
       * The time after the workflow run was finished
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      finishedAfter?: string;
      /**
       * The time before the workflow run was finished
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      finishedBefore?: string;
      /** The order by field */
      orderByField?: WorkflowRunOrderByField;
      /** The order by direction */
      orderByDirection?: WorkflowRunOrderByDirection;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRunList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/runs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Replays a list of workflow runs.
   *
   * @tags Workflow Run
   * @name WorkflowRunUpdateReplay
   * @summary Replay workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflow-runs/replay
   * @secure
   */
  workflowRunUpdateReplay = Object.assign((
    tenant: string,
    data: ReplayWorkflowRunsRequest,
    params: RequestParams = {},
  ) =>
    this.request<ReplayWorkflowRunsResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/replay`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a summary of  workflow run metrics for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGetMetrics
   * @summary Get workflow runs metrics
   * @request GET:/api/v1/tenants/{tenant}/workflows/runs/metrics
   * @secure
   */
  workflowRunGetMetrics = Object.assign((
    tenant: string,
    query?: {
      /**
       * The event id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      eventId?: string;
      /**
       * The workflow id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      workflowId?: string;
      /**
       * The parent workflow run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentWorkflowRunId?: string;
      /**
       * The parent step run id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      parentStepRunId?: string;
      /**
       * A list of metadata key value pairs to filter by
       * @example ["key1:value1","key2:value2"]
       */
      additionalMetadata?: string[];
      /**
       * The time after the workflow run was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdAfter?: string;
      /**
       * The time before the workflow run was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdBefore?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRunsMetrics, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/runs/metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGet
   * @summary Get workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}
   * @secure
   */
  workflowRunGet = Object.assign((
    tenant: string,
    workflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow-run"],
    }), { resources: new Set<string>(["tenant", "workflow-run"]) });
  /**
   * @description Get a workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGetShape
   * @summary Get workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/shape
   * @secure
   */
  workflowRunGetShape = Object.assign((
    tenant: string,
    workflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRunShape, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}/shape`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow-run"],
    }), { resources: new Set<string>(["tenant", "workflow-run"]) });
  /**
   * @description Get a step run by id
   *
   * @tags Step Run
   * @name StepRunGet
   * @summary Get step run
   * @request GET:/api/v1/tenants/{tenant}/step-runs/{step-run}
   * @secure
   */
  stepRunGet = Object.assign((tenant: string, stepRun: string, params: RequestParams = {}) =>
    this.request<StepRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description Reruns a step run
   *
   * @tags Step Run
   * @name StepRunUpdateRerun
   * @summary Rerun step run
   * @request POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/rerun
   * @secure
   */
  stepRunUpdateRerun = Object.assign((
    tenant: string,
    stepRun: string,
    data: RerunStepRunRequest,
    params: RequestParams = {},
  ) =>
    this.request<StepRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}/rerun`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description Attempts to cancel a step run
   *
   * @tags Step Run
   * @name StepRunUpdateCancel
   * @summary Attempts to cancel a step run
   * @request POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/cancel
   * @secure
   */
  stepRunUpdateCancel = Object.assign((
    tenant: string,
    stepRun: string,
    params: RequestParams = {},
  ) =>
    this.request<StepRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}/cancel`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description Get the schema for a step run
   *
   * @tags Step Run
   * @name StepRunGetSchema
   * @summary Get step run schema
   * @request GET:/api/v1/tenants/{tenant}/step-runs/{step-run}/schema
   * @secure
   */
  stepRunGetSchema = Object.assign((
    tenant: string,
    stepRun: string,
    params: RequestParams = {},
  ) =>
    this.request<object, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}/schema`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "step-run"],
    }), { resources: new Set<string>(["tenant", "step-run"]) });
  /**
   * @description Get all workers for a tenant
   *
   * @tags Worker
   * @name WorkerList
   * @summary Get workers
   * @request GET:/api/v1/tenants/{tenant}/worker
   * @secure
   */
  workerList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<WorkerList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/worker`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Update a worker
   *
   * @tags Worker
   * @name WorkerUpdate
   * @summary Update worker
   * @request PATCH:/api/v1/workers/{worker}
   * @secure
   */
  workerUpdate = Object.assign((
    worker: string,
    data: UpdateWorkerRequest,
    params: RequestParams = {},
  ) =>
    this.request<Worker, APIErrors>({
      path: `/api/v1/workers/${worker}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "worker"],
    }), { resources: new Set<string>(["tenant", "worker"]) });
  /**
   * @description Get a worker
   *
   * @tags Worker
   * @name WorkerGet
   * @summary Get worker
   * @request GET:/api/v1/workers/{worker}
   * @secure
   */
  workerGet = Object.assign((worker: string, params: RequestParams = {}) =>
    this.request<Worker, APIErrors>({
      path: `/api/v1/workers/${worker}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "worker"],
    }), { resources: new Set<string>(["tenant", "worker"]) });
  /**
   * @description Lists all webhooks
   *
   * @name WebhookList
   * @summary List webhooks
   * @request GET:/api/v1/tenants/{tenant}/webhook-workers
   * @secure
   */
  webhookList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<WebhookWorkerListResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/webhook-workers`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Creates a webhook
   *
   * @name WebhookCreate
   * @summary Create a webhook
   * @request POST:/api/v1/tenants/{tenant}/webhook-workers
   * @secure
   */
  webhookCreate = Object.assign((
    tenant: string,
    data: WebhookWorkerCreateRequest,
    params: RequestParams = {},
  ) =>
    this.request<WebhookWorkerCreated, APIErrors>({
      path: `/api/v1/tenants/${tenant}/webhook-workers`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Deletes a webhook
   *
   * @name WebhookDelete
   * @summary Delete a webhook
   * @request DELETE:/api/v1/webhook-workers/{webhook}
   * @secure
   */
  webhookDelete = Object.assign((webhook: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/webhook-workers/${webhook}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["tenant", "webhook"],
    }), { resources: new Set<string>(["tenant", "webhook"]) });
  /**
   * @description Lists all requests for a webhook
   *
   * @name WebhookRequestsList
   * @summary List webhook requests
   * @request GET:/api/v1/webhook-workers/{webhook}/requests
   * @secure
   */
  webhookRequestsList = Object.assign((webhook: string, params: RequestParams = {}) =>
    this.request<WebhookWorkerRequestListResponse, APIErrors>({
      path: `/api/v1/webhook-workers/${webhook}/requests`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "webhook"],
    }), { resources: new Set<string>(["tenant", "webhook"]) });
  /**
   * @description Get the input for a workflow run.
   *
   * @tags Workflow Run
   * @name WorkflowRunGetInput
   * @summary Get workflow run input
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/input
   * @secure
   */
  workflowRunGetInput = Object.assign((
    tenant: string,
    workflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<Record<string, any>, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}/input`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "workflow-run"],
    }), { resources: new Set<string>(["tenant", "workflow-run"]) });
  /**
   * @description Triggers a workflow to check the status of the instance
   *
   * @name MonitoringPostRunProbe
   * @summary Detailed Health Probe For the Instance
   * @request POST:/api/v1/monitoring/{tenant}/probe
   * @secure
   */
  monitoringPostRunProbe = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/monitoring/${tenant}/probe`,
      method: "POST",
      secure: true,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get the version of the server
   *
   * @name InfoGetVersion
   * @summary We return the version for the currently running server
   * @request GET:/api/v1/version
   */
  infoGetVersion = Object.assign((params: RequestParams = {}) =>
    this.request<
      {
        /** @example "1.0.0" */
        version: string;
      },
      any
    >({
      path: `/api/v1/version`,
      method: "GET",
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Get the prometheus metrics for the tenant
   *
   * @tags Tenant
   * @name TenantGetPrometheusMetrics
   * @summary Get prometheus metrics
   * @request GET:/api/v1/tenants/{tenant}/prometheus-metrics
   * @secure
   */
  tenantGetPrometheusMetrics = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<EventKey, APIErrors>({
      path: `/api/v1/tenants/${tenant}/prometheus-metrics`,
      method: "GET",
      secure: true,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get task stats for tenant
   *
   * @tags Tenant
   * @name TenantGetTaskStats
   * @summary Get task stats for tenant
   * @request GET:/api/v1/tenants/{tenant}/task-stats
   * @secure
   */
  tenantGetTaskStats = Object.assign((
    tenant: string,
    query?: {
      /** Task names that must appear in the response. Missing tasks are zero-filled so KEDA's metrics-api JSONPath always resolves. */
      taskNames?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<TaskStats, APIErrors>({
      path: `/api/v1/tenants/${tenant}/task-stats`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Evaluate a feature flag for a tenant
   *
   * @tags Feature Flags
   * @name TenantFeatureFlagEvaluate
   * @summary Evaluate a feature flag for a tenant
   * @request GET:/api/v1/tenants/{tenant}/feature-flags
   * @secure
   */
  tenantFeatureFlagEvaluate = Object.assign((
    tenant: string,
    query: {
      /** The feature flag id to evaluate */
      featureFlagId: FeatureFlagId;
      /** A flag indicating what the behavior of the feature flag should be if PostHog is disabled or unavailable */
      isEnabledIfNoPosthog: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<FeatureFlagEvaluationResult, APIErrors>({
      path: `/api/v1/tenants/${tenant}/feature-flags`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
}
