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
  ListAPIMetaIntegration,
  ListAPITokensResponse,
  ListSNSIntegrations,
  ListSlackWebhooks,
  LogLineLevelField,
  LogLineList,
  LogLineOrderByDirection,
  LogLineOrderByField,
  LogLineSearch,
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
  ScheduledWorkflowsList,
  ScheduledWorkflowsOrderByField,
  StepRun,
  StepRunArchiveList,
  StepRunEventList,
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
  UpdateTenantAlertEmailGroupRequest,
  UpdateTenantInviteRequest,
  UpdateTenantRequest,
  UpdateWorkerRequest,
  User,
  UserChangePasswordRequest,
  UserLoginRequest,
  UserRegisterRequest,
  UserTenantMembershipsList,
  V1CancelTaskRequest,
  V1CreateFilterRequest,
  V1DagChildren,
  V1EventList,
  V1Filter,
  V1FilterList,
  V1LogLineList,
  V1ReplayTaskRequest,
  V1TaskEventList,
  V1TaskPointMetrics,
  V1TaskRunMetrics,
  V1TaskStatus,
  V1TaskSummary,
  V1TaskSummaryList,
  V1TaskTimingList,
  V1TriggerWorkflowRunRequest,
  V1WorkflowRunDetails,
  V1WorkflowRunDisplayNameList,
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
  v1TaskGet = (
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
    });
  /**
   * @description List events for a task
   *
   * @tags Task
   * @name V1TaskEventList
   * @summary List events for a task
   * @request GET:/api/v1/stable/tasks/{task}/task-events
   * @secure
   */
  v1TaskEventList = (
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
    });
  /**
   * @description Lists log lines for a task
   *
   * @tags Log
   * @name V1LogLineList
   * @summary List log lines
   * @request GET:/api/v1/stable/tasks/{task}/logs
   * @secure
   */
  v1LogLineList = (task: string, params: RequestParams = {}) =>
    this.request<V1LogLineList, APIErrors>({
      path: `/api/v1/stable/tasks/${task}/logs`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Cancel tasks
   *
   * @tags Task
   * @name V1TaskCancel
   * @summary Cancel tasks
   * @request POST:/api/v1/stable/tenants/{tenant}/tasks/cancel
   * @secure
   */
  v1TaskCancel = (
    tenant: string,
    data: V1CancelTaskRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/tasks/cancel`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Replay tasks
   *
   * @tags Task
   * @name V1TaskReplay
   * @summary Replay tasks
   * @request POST:/api/v1/stable/tenants/{tenant}/tasks/replay
   * @secure
   */
  v1TaskReplay = (
    tenant: string,
    data: V1ReplayTaskRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/stable/tenants/${tenant}/tasks/replay`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Lists all tasks that belong a specific list of dags
   *
   * @tags Task
   * @name V1DagListTasks
   * @summary List tasks
   * @request GET:/api/v1/stable/dags/tasks
   * @secure
   */
  v1DagListTasks = (
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
    });
  /**
   * @description Lists workflow runs for a tenant.
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunList
   * @summary List workflow runs
   * @request GET:/api/v1/stable/tenants/{tenant}/workflow-runs
   * @secure
   */
  v1WorkflowRunList = (
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
    });
  /**
   * @description Lists displayable names of workflow runs for a tenant
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunDisplayNamesList
   * @summary List workflow runs
   * @request GET:/api/v1/stable/tenants/{tenant}/workflow-runs/display-names
   * @secure
   */
  v1WorkflowRunDisplayNamesList = (
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
    });
  /**
   * @description Trigger a new workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunCreate
   * @summary Create workflow run
   * @request POST:/api/v1/stable/tenants/{tenant}/workflow-runs/trigger
   * @secure
   */
  v1WorkflowRunCreate = (
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
    });
  /**
   * @description Get a workflow run and its metadata to display on the "detail" page
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunGet
   * @summary List tasks
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}
   * @secure
   */
  v1WorkflowRunGet = (v1WorkflowRun: string, params: RequestParams = {}) =>
    this.request<V1WorkflowRunDetails, APIErrors>({
      path: `/api/v1/stable/workflow-runs/${v1WorkflowRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description List all tasks for a workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunTaskEventsList
   * @summary List tasks
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/task-events
   * @secure
   */
  v1WorkflowRunTaskEventsList = (
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
    });
  /**
   * @description Get the timings for a workflow run
   *
   * @tags Workflow Runs
   * @name V1WorkflowRunGetTimings
   * @summary List timings for a workflow run
   * @request GET:/api/v1/stable/workflow-runs/{v1-workflow-run}/task-timings
   * @secure
   */
  v1WorkflowRunGetTimings = (
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
    });
  /**
   * @description Get a summary of task run metrics for a tenant
   *
   * @tags Task
   * @name V1TaskListStatusMetrics
   * @summary Get task metrics
   * @request GET:/api/v1/stable/tenants/{tenant}/task-metrics
   * @secure
   */
  v1TaskListStatusMetrics = (
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
    });
  /**
   * @description Get a minute by minute breakdown of task metrics for a tenant
   *
   * @tags Task
   * @name V1TaskGetPointMetrics
   * @summary Get task point metrics
   * @request GET:/api/v1/stable/tenants/{tenant}/task-point-metrics
   * @secure
   */
  v1TaskGetPointMetrics = (
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
    });
  /**
   * @description Lists all events for a tenant.
   *
   * @tags Event
   * @name V1EventList
   * @summary List events
   * @request GET:/api/v1/stable/tenants/{tenant}/events
   * @secure
   */
  v1EventList = (
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
    });
  /**
   * @description Lists all filters for a tenant.
   *
   * @tags Filter
   * @name V1FilterList
   * @summary List filters
   * @request GET:/api/v1/stable/tenants/{tenant}/filters
   * @secure
   */
  v1FilterList = (
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
    });
  /**
   * @description Create a new filter
   *
   * @tags Filter
   * @name V1FilterCreate
   * @summary Create a filter
   * @request POST:/api/v1/stable/tenants/{tenant}/filters
   * @secure
   */
  v1FilterCreate = (
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
    });
  /**
   * @description Get a filter by its id
   *
   * @tags Filter
   * @name V1FilterGet
   * @summary Get a filter
   * @request GET:/api/v1/stable/tenants/{tenant}/filters/{v1-filter}
   * @secure
   */
  v1FilterGet = (
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
    });
  /**
   * @description Delete a filter
   *
   * @tags Filter
   * @name V1FilterDelete
   * @request DELETE:/api/v1/stable/tenants/{tenant}/filters/{v1-filter}
   * @secure
   */
  v1FilterDelete = (
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
    });
  /**
   * @description Gets the readiness status
   *
   * @tags Healthcheck
   * @name ReadinessGet
   * @summary Get readiness
   * @request GET:/api/ready
   */
  readinessGet = (params: RequestParams = {}) =>
    this.request<void, void>({
      path: `/api/ready`,
      method: "GET",
      ...params,
    });
  /**
   * @description Gets the liveness status
   *
   * @tags Healthcheck
   * @name LivenessGet
   * @summary Get liveness
   * @request GET:/api/live
   */
  livenessGet = (params: RequestParams = {}) =>
    this.request<void, void>({
      path: `/api/live`,
      method: "GET",
      ...params,
    });
  /**
   * @description Gets metadata for the Hatchet instance
   *
   * @tags Metadata
   * @name MetadataGet
   * @summary Get metadata
   * @request GET:/api/v1/meta
   */
  metadataGet = (params: RequestParams = {}) =>
    this.request<APIMeta, APIErrors>({
      path: `/api/v1/meta`,
      method: "GET",
      format: "json",
      ...params,
    });
  /**
   * @description Gets metadata for the Hatchet cloud instance
   *
   * @tags Metadata
   * @name CloudMetadataGet
   * @summary Get cloud metadata
   * @request GET:/api/v1/cloud/metadata
   */
  cloudMetadataGet = (params: RequestParams = {}) =>
    this.request<APIErrors, APIErrors>({
      path: `/api/v1/cloud/metadata`,
      method: "GET",
      format: "json",
      ...params,
    });
  /**
   * @description List all integrations
   *
   * @tags Metadata
   * @name MetadataListIntegrations
   * @summary List integrations
   * @request GET:/api/v1/meta/integrations
   * @secure
   */
  metadataListIntegrations = (params: RequestParams = {}) =>
    this.request<ListAPIMetaIntegration, APIErrors>({
      path: `/api/v1/meta/integrations`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Logs in a user.
   *
   * @tags User
   * @name UserUpdateLogin
   * @summary Login user
   * @request POST:/api/v1/users/login
   */
  userUpdateLogin = (data: UserLoginRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/login`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGoogleOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/users/google/start
   */
  userUpdateGoogleOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/google/start`,
      method: "GET",
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGoogleOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/google/callback
   */
  userUpdateGoogleOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/google/callback`,
      method: "GET",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/users/github/start
   */
  userUpdateGithubOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/start`,
      method: "GET",
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/github/callback
   */
  userUpdateGithubOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/callback`,
      method: "GET",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateSlackOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/tenants/{tenant}/slack/start
   * @secure
   */
  userUpdateSlackOauthStart = (tenant: string, params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/tenants/${tenant}/slack/start`,
      method: "GET",
      secure: true,
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateSlackOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/slack/callback
   * @secure
   */
  userUpdateSlackOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/slack/callback`,
      method: "GET",
      secure: true,
      ...params,
    });
  /**
   * @description SNS event
   *
   * @tags Github
   * @name SnsUpdate
   * @summary Github app tenant webhook
   * @request POST:/api/v1/sns/{tenant}/{event}
   */
  snsUpdate = (tenant: string, event: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/sns/${tenant}/${event}`,
      method: "POST",
      ...params,
    });
  /**
   * @description List SNS integrations
   *
   * @tags SNS
   * @name SnsList
   * @summary List SNS integrations
   * @request GET:/api/v1/tenants/{tenant}/sns
   * @secure
   */
  snsList = (tenant: string, params: RequestParams = {}) =>
    this.request<ListSNSIntegrations, APIErrors>({
      path: `/api/v1/tenants/${tenant}/sns`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create SNS integration
   *
   * @tags SNS
   * @name SnsCreate
   * @summary Create SNS integration
   * @request POST:/api/v1/tenants/{tenant}/sns
   * @secure
   */
  snsCreate = (
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
    });
  /**
   * @description Creates a new tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupCreate
   * @summary Create tenant alert email group
   * @request POST:/api/v1/tenants/{tenant}/alerting-email-groups
   * @secure
   */
  alertEmailGroupCreate = (
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
    });
  /**
   * @description Gets a list of tenant alert email groups
   *
   * @tags Tenant
   * @name AlertEmailGroupList
   * @summary List tenant alert email groups
   * @request GET:/api/v1/tenants/{tenant}/alerting-email-groups
   * @secure
   */
  alertEmailGroupList = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantAlertEmailGroupList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/alerting-email-groups`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Gets the resource policy for a tenant
   *
   * @tags Tenant
   * @name TenantResourcePolicyGet
   * @summary Create tenant alert email group
   * @request GET:/api/v1/tenants/{tenant}/resource-policy
   * @secure
   */
  tenantResourcePolicyGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantResourcePolicy, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/resource-policy`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Updates a tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupUpdate
   * @summary Update tenant alert email group
   * @request PATCH:/api/v1/alerting-email-groups/{alert-email-group}
   * @secure
   */
  alertEmailGroupUpdate = (
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
    });
  /**
   * @description Deletes a tenant alert email group
   *
   * @tags Tenant
   * @name AlertEmailGroupDelete
   * @summary Delete tenant alert email group
   * @request DELETE:/api/v1/alerting-email-groups/{alert-email-group}
   * @secure
   */
  alertEmailGroupDelete = (
    alertEmailGroup: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/alerting-email-groups/${alertEmailGroup}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Delete SNS integration
   *
   * @tags SNS
   * @name SnsDelete
   * @summary Delete SNS integration
   * @request DELETE:/api/v1/sns/{sns}
   * @secure
   */
  snsDelete = (sns: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/sns/${sns}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List Slack webhooks
   *
   * @tags Slack
   * @name SlackWebhookList
   * @summary List Slack integrations
   * @request GET:/api/v1/tenants/{tenant}/slack
   * @secure
   */
  slackWebhookList = (tenant: string, params: RequestParams = {}) =>
    this.request<ListSlackWebhooks, APIErrors>({
      path: `/api/v1/tenants/${tenant}/slack`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Delete Slack webhook
   *
   * @tags Slack
   * @name SlackWebhookDelete
   * @summary Delete Slack webhook
   * @request DELETE:/api/v1/slack/{slack}
   * @secure
   */
  slackWebhookDelete = (slack: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/slack/${slack}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Gets the current user
   *
   * @tags User
   * @name UserGetCurrent
   * @summary Get current user
   * @request GET:/api/v1/users/current
   * @secure
   */
  userGetCurrent = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/current`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a user password.
   *
   * @tags User
   * @name UserUpdatePassword
   * @summary Change user password
   * @request POST:/api/v1/users/password
   * @secure
   */
  userUpdatePassword = (
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
    });
  /**
   * @description Registers a user.
   *
   * @tags User
   * @name UserCreate
   * @summary Register user
   * @request POST:/api/v1/users/register
   */
  userCreate = (data: UserRegisterRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/register`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Logs out a user.
   *
   * @tags User
   * @name UserUpdateLogout
   * @summary Logout user
   * @request POST:/api/v1/users/logout
   * @secure
   */
  userUpdateLogout = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/logout`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all tenant memberships for the current user
   *
   * @tags User
   * @name TenantMembershipsList
   * @summary List tenant memberships
   * @request GET:/api/v1/users/memberships
   * @secure
   */
  tenantMembershipsList = (params: RequestParams = {}) =>
    this.request<UserTenantMembershipsList, APIErrors>({
      path: `/api/v1/users/memberships`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all tenant invites for the current user
   *
   * @tags Tenant
   * @name UserListTenantInvites
   * @summary List tenant invites
   * @request GET:/api/v1/users/invites
   * @secure
   */
  userListTenantInvites = (params: RequestParams = {}) =>
    this.request<TenantInviteList, APIErrors>({
      path: `/api/v1/users/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Accepts a tenant invite
   *
   * @tags Tenant
   * @name TenantInviteAccept
   * @summary Accept tenant invite
   * @request POST:/api/v1/users/invites/accept
   * @secure
   */
  tenantInviteAccept = (
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
    });
  /**
   * @description Rejects a tenant invite
   *
   * @tags Tenant
   * @name TenantInviteReject
   * @summary Reject tenant invite
   * @request POST:/api/v1/users/invites/reject
   * @secure
   */
  tenantInviteReject = (
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
    });
  /**
   * @description Creates a new tenant
   *
   * @tags Tenant
   * @name TenantCreate
   * @summary Create tenant
   * @request POST:/api/v1/tenants
   * @secure
   */
  tenantCreate = (data: CreateTenantRequest, params: RequestParams = {}) =>
    this.request<Tenant, APIErrors | APIError>({
      path: `/api/v1/tenants`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Update an existing tenant
   *
   * @tags Tenant
   * @name TenantUpdate
   * @summary Update tenant
   * @request PATCH:/api/v1/tenants/{tenant}
   * @secure
   */
  tenantUpdate = (
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
    });
  /**
   * @description Gets the alerting settings for a tenant
   *
   * @tags Tenant
   * @name TenantAlertingSettingsGet
   * @summary Get tenant alerting settings
   * @request GET:/api/v1/tenants/{tenant}/alerting/settings
   * @secure
   */
  tenantAlertingSettingsGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantAlertingSettings, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/alerting/settings`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Creates a new tenant invite
   *
   * @tags Tenant
   * @name TenantInviteCreate
   * @summary Create tenant invite
   * @request POST:/api/v1/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteCreate = (
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
    });
  /**
   * @description Gets a list of tenant invites
   *
   * @tags Tenant
   * @name TenantInviteList
   * @summary List tenant invites
   * @request GET:/api/v1/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteList = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantInviteList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Updates a tenant invite
   *
   * @name TenantInviteUpdate
   * @summary Update invite
   * @request PATCH:/api/v1/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteUpdate = (
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
    });
  /**
   * @description Deletes a tenant invite
   *
   * @name TenantInviteDelete
   * @summary Delete invite
   * @request DELETE:/api/v1/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteDelete = (
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
    });
  /**
   * @description Create an API token for a tenant
   *
   * @tags API Token
   * @name ApiTokenCreate
   * @summary Create API Token
   * @request POST:/api/v1/tenants/{tenant}/api-tokens
   * @secure
   */
  apiTokenCreate = (
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
    });
  /**
   * @description List API tokens for a tenant
   *
   * @tags API Token
   * @name ApiTokenList
   * @summary List API Tokens
   * @request GET:/api/v1/tenants/{tenant}/api-tokens
   * @secure
   */
  apiTokenList = (tenant: string, params: RequestParams = {}) =>
    this.request<ListAPITokensResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/api-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Revoke an API token for a tenant
   *
   * @tags API Token
   * @name ApiTokenUpdateRevoke
   * @summary Revoke API Token
   * @request POST:/api/v1/api-tokens/{api-token}
   * @secure
   */
  apiTokenUpdateRevoke = (apiToken: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/api-tokens/${apiToken}`,
      method: "POST",
      secure: true,
      ...params,
    });
  /**
   * @description Get the queue metrics for the tenant
   *
   * @tags Workflow
   * @name TenantGetQueueMetrics
   * @summary Get workflow metrics
   * @request GET:/api/v1/tenants/{tenant}/queue-metrics
   * @secure
   */
  tenantGetQueueMetrics = (
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
    });
  /**
   * @description Get the queue metrics for the tenant
   *
   * @tags Tenant
   * @name TenantGetStepRunQueueMetrics
   * @summary Get step run metrics
   * @request GET:/api/v1/tenants/{tenant}/step-run-queue-metrics
   * @secure
   */
  tenantGetStepRunQueueMetrics = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantStepRunQueueMetrics, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-run-queue-metrics`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all events for a tenant.
   *
   * @tags Event
   * @name EventList
   * @summary List events
   * @request GET:/api/v1/tenants/{tenant}/events
   * @secure
   */
  eventList = (
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
    });
  /**
   * @description Creates a new event.
   *
   * @tags Event
   * @name EventCreate
   * @summary Create event
   * @request POST:/api/v1/tenants/{tenant}/events
   * @secure
   */
  eventCreate = (
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
    });
  /**
   * @description Bulk creates new events.
   *
   * @tags Event
   * @name EventCreateBulk
   * @summary Bulk Create events
   * @request POST:/api/v1/tenants/{tenant}/events/bulk
   * @secure
   */
  eventCreateBulk = (
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
    });
  /**
   * @description Replays a list of events.
   *
   * @tags Event
   * @name EventUpdateReplay
   * @summary Replay events
   * @request POST:/api/v1/tenants/{tenant}/events/replay
   * @secure
   */
  eventUpdateReplay = (
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
    });
  /**
   * @description Cancels all runs for a list of events.
   *
   * @tags Event
   * @name EventUpdateCancel
   * @summary Replay events
   * @request POST:/api/v1/tenants/{tenant}/events/cancel
   * @secure
   */
  eventUpdateCancel = (
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
    });
  /**
   * @description Lists all rate limits for a tenant.
   *
   * @tags Rate Limits
   * @name RateLimitList
   * @summary List rate limits
   * @request GET:/api/v1/tenants/{tenant}/rate-limits
   * @secure
   */
  rateLimitList = (
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
    });
  /**
   * @description Gets a list of tenant members
   *
   * @tags Tenant
   * @name TenantMemberList
   * @summary List tenant members
   * @request GET:/api/v1/tenants/{tenant}/members
   * @secure
   */
  tenantMemberList = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantMemberList, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/members`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Delete a member from a tenant
   *
   * @tags Tenant
   * @name TenantMemberDelete
   * @summary Delete a tenant member
   * @request DELETE:/api/v1/tenants/{tenant}/members/{member}
   * @secure
   */
  tenantMemberDelete = (
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
    });
  /**
   * @description Get an event.
   *
   * @tags Event
   * @name EventGet
   * @summary Get event data
   * @request GET:/api/v1/events/{event}
   * @secure
   */
  eventGet = (event: string, params: RequestParams = {}) =>
    this.request<Event, APIErrors>({
      path: `/api/v1/events/${event}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get the data for an event.
   *
   * @tags Event
   * @name EventDataGet
   * @summary Get event data
   * @request GET:/api/v1/events/{event}/data
   * @secure
   */
  eventDataGet = (event: string, params: RequestParams = {}) =>
    this.request<EventData, APIErrors>({
      path: `/api/v1/events/${event}/data`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all event keys for a tenant.
   *
   * @tags Event
   * @name EventKeyList
   * @summary List event keys
   * @request GET:/api/v1/tenants/{tenant}/events/keys
   * @secure
   */
  eventKeyList = (tenant: string, params: RequestParams = {}) =>
    this.request<EventKeyList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/keys`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all workflows for a tenant
   *
   * @tags Workflow
   * @name WorkflowList
   * @summary Get workflows
   * @request GET:/api/v1/tenants/{tenant}/workflows
   * @secure
   */
  workflowList = (
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
    });
  /**
   * @description Schedule a new workflow run for a tenant
   *
   * @tags Workflow Run
   * @name ScheduledWorkflowRunCreate
   * @summary Trigger workflow run
   * @request POST:/api/v1/tenants/{tenant}/workflows/{workflow}/scheduled
   * @secure
   */
  scheduledWorkflowRunCreate = (
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
    });
  /**
   * @description Get all scheduled workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledList
   * @summary Get scheduled workflow runs
   * @request GET:/api/v1/tenants/{tenant}/workflows/scheduled
   * @secure
   */
  workflowScheduledList = (
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
    });
  /**
   * @description Get a scheduled workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledGet
   * @summary Get scheduled workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}
   * @secure
   */
  workflowScheduledGet = (
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
    });
  /**
   * @description Delete a scheduled workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowScheduledDelete
   * @summary Delete scheduled workflow run
   * @request DELETE:/api/v1/tenants/{tenant}/workflows/scheduled/{scheduled-workflow-run}
   * @secure
   */
  workflowScheduledDelete = (
    tenant: string,
    scheduledWorkflowRun: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/workflows/scheduled/${scheduledWorkflowRun}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Create a new cron job workflow trigger for a tenant
   *
   * @tags Workflow Run
   * @name CronWorkflowTriggerCreate
   * @summary Create cron job workflow trigger
   * @request POST:/api/v1/tenants/{tenant}/workflows/{workflow}/crons
   * @secure
   */
  cronWorkflowTriggerCreate = (
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
    });
  /**
   * @description Get all cron job workflow triggers for a tenant
   *
   * @tags Workflow
   * @name CronWorkflowList
   * @summary Get cron job workflows
   * @request GET:/api/v1/tenants/{tenant}/workflows/crons
   * @secure
   */
  cronWorkflowList = (
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
    });
  /**
   * @description Get a cron job workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowCronGet
   * @summary Get cron job workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}
   * @secure
   */
  workflowCronGet = (
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
    });
  /**
   * @description Delete a cron job workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowCronDelete
   * @summary Delete cron job workflow run
   * @request DELETE:/api/v1/tenants/{tenant}/workflows/crons/{cron-workflow}
   * @secure
   */
  workflowCronDelete = (
    tenant: string,
    cronWorkflow: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/workflows/crons/${cronWorkflow}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Cancel a batch of workflow runs
   *
   * @tags Workflow Run
   * @name WorkflowRunCancel
   * @summary Cancel workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflows/cancel
   * @secure
   */
  workflowRunCancel = (
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
    });
  /**
   * @description Get a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowGet
   * @summary Get workflow
   * @request GET:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowGet = (workflow: string, params: RequestParams = {}) =>
    this.request<Workflow, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Delete a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowDelete
   * @summary Delete workflow
   * @request DELETE:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowDelete = (workflow: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Update a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowUpdate
   * @summary Update workflow
   * @request PATCH:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowUpdate = (
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
    });
  /**
   * @description Get a workflow version for a tenant
   *
   * @tags Workflow
   * @name WorkflowVersionGet
   * @summary Get workflow version
   * @request GET:/api/v1/workflows/{workflow}/versions
   * @secure
   */
  workflowVersionGet = (
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
    });
  /**
   * @description Trigger a new workflow run for a tenant
   *
   * @tags Workflow Run
   * @name WorkflowRunCreate
   * @summary Trigger workflow run
   * @request POST:/api/v1/workflows/{workflow}/trigger
   * @secure
   */
  workflowRunCreate = (
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
    });
  /**
   * @description Get the metrics for a workflow version
   *
   * @tags Workflow
   * @name WorkflowGetMetrics
   * @summary Get workflow metrics
   * @request GET:/api/v1/workflows/{workflow}/metrics
   * @secure
   */
  workflowGetMetrics = (
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
    });
  /**
   * @description Lists log lines for a step run.
   *
   * @tags Log
   * @name LogLineList
   * @summary List log lines
   * @request GET:/api/v1/step-runs/{step-run}/logs
   * @secure
   */
  logLineList = (
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
      /** A list of levels to filter by */
      levels?: LogLineLevelField;
      /** The search query to filter for */
      search?: LogLineSearch;
      /** What to order by */
      orderByField?: LogLineOrderByField;
      /** The order direction */
      orderByDirection?: LogLineOrderByDirection;
    },
    params: RequestParams = {},
  ) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description List events for a step run
   *
   * @tags Step Run
   * @name StepRunListEvents
   * @summary List events for step run
   * @request GET:/api/v1/step-runs/{step-run}/events
   * @secure
   */
  stepRunListEvents = (
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
    });
  /**
   * @description List events for all step runs for a workflow run
   *
   * @tags Step Run
   * @name WorkflowRunListStepRunEvents
   * @summary List events for all step runs for a workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/step-run-events
   * @secure
   */
  workflowRunListStepRunEvents = (
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
    });
  /**
   * @description List archives for a step run
   *
   * @tags Step Run
   * @name StepRunListArchives
   * @summary List archives for step run
   * @request GET:/api/v1/step-runs/{step-run}/archives
   * @secure
   */
  stepRunListArchives = (
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
    });
  /**
   * @description Get a count of the workers available for workflow
   *
   * @tags Workflow
   * @name WorkflowGetWorkersCount
   * @summary Get workflow worker count
   * @request GET:/api/v1/tenants/{tenant}/workflows/{workflow}/worker-count
   * @secure
   */
  workflowGetWorkersCount = (
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
    });
  /**
   * @description Get all workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunList
   * @summary Get workflow runs
   * @request GET:/api/v1/tenants/{tenant}/workflows/runs
   * @secure
   */
  workflowRunList = (
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
    });
  /**
   * @description Replays a list of workflow runs.
   *
   * @tags Workflow Run
   * @name WorkflowRunUpdateReplay
   * @summary Replay workflow runs
   * @request POST:/api/v1/tenants/{tenant}/workflow-runs/replay
   * @secure
   */
  workflowRunUpdateReplay = (
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
    });
  /**
   * @description Get a summary of  workflow run metrics for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGetMetrics
   * @summary Get workflow runs metrics
   * @request GET:/api/v1/tenants/{tenant}/workflows/runs/metrics
   * @secure
   */
  workflowRunGetMetrics = (
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
    });
  /**
   * @description Get a workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGet
   * @summary Get workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}
   * @secure
   */
  workflowRunGet = (
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
    });
  /**
   * @description Get a workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGetShape
   * @summary Get workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/shape
   * @secure
   */
  workflowRunGetShape = (
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
    });
  /**
   * @description Get a step run by id
   *
   * @tags Step Run
   * @name StepRunGet
   * @summary Get step run
   * @request GET:/api/v1/tenants/{tenant}/step-runs/{step-run}
   * @secure
   */
  stepRunGet = (tenant: string, stepRun: string, params: RequestParams = {}) =>
    this.request<StepRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Reruns a step run
   *
   * @tags Step Run
   * @name StepRunUpdateRerun
   * @summary Rerun step run
   * @request POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/rerun
   * @secure
   */
  stepRunUpdateRerun = (
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
    });
  /**
   * @description Attempts to cancel a step run
   *
   * @tags Step Run
   * @name StepRunUpdateCancel
   * @summary Attempts to cancel a step run
   * @request POST:/api/v1/tenants/{tenant}/step-runs/{step-run}/cancel
   * @secure
   */
  stepRunUpdateCancel = (
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
    });
  /**
   * @description Get the schema for a step run
   *
   * @tags Step Run
   * @name StepRunGetSchema
   * @summary Get step run schema
   * @request GET:/api/v1/tenants/{tenant}/step-runs/{step-run}/schema
   * @secure
   */
  stepRunGetSchema = (
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
    });
  /**
   * @description Get all workers for a tenant
   *
   * @tags Worker
   * @name WorkerList
   * @summary Get workers
   * @request GET:/api/v1/tenants/{tenant}/worker
   * @secure
   */
  workerList = (tenant: string, params: RequestParams = {}) =>
    this.request<WorkerList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/worker`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a worker
   *
   * @tags Worker
   * @name WorkerUpdate
   * @summary Update worker
   * @request PATCH:/api/v1/workers/{worker}
   * @secure
   */
  workerUpdate = (
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
    });
  /**
   * @description Get a worker
   *
   * @tags Worker
   * @name WorkerGet
   * @summary Get worker
   * @request GET:/api/v1/workers/{worker}
   * @secure
   */
  workerGet = (worker: string, params: RequestParams = {}) =>
    this.request<Worker, APIErrors>({
      path: `/api/v1/workers/${worker}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all webhooks
   *
   * @name WebhookList
   * @summary List webhooks
   * @request GET:/api/v1/tenants/{tenant}/webhook-workers
   * @secure
   */
  webhookList = (tenant: string, params: RequestParams = {}) =>
    this.request<WebhookWorkerListResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/webhook-workers`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Creates a webhook
   *
   * @name WebhookCreate
   * @summary Create a webhook
   * @request POST:/api/v1/tenants/{tenant}/webhook-workers
   * @secure
   */
  webhookCreate = (
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
    });
  /**
   * @description Deletes a webhook
   *
   * @name WebhookDelete
   * @summary Delete a webhook
   * @request DELETE:/api/v1/webhook-workers/{webhook}
   * @secure
   */
  webhookDelete = (webhook: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/webhook-workers/${webhook}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Lists all requests for a webhook
   *
   * @name WebhookRequestsList
   * @summary List webhook requests
   * @request GET:/api/v1/webhook-workers/{webhook}/requests
   * @secure
   */
  webhookRequestsList = (webhook: string, params: RequestParams = {}) =>
    this.request<WebhookWorkerRequestListResponse, APIErrors>({
      path: `/api/v1/webhook-workers/${webhook}/requests`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get the input for a workflow run.
   *
   * @tags Workflow Run
   * @name WorkflowRunGetInput
   * @summary Get workflow run input
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/input
   * @secure
   */
  workflowRunGetInput = (
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
    });
  /**
   * @description Triggers a workflow to check the status of the instance
   *
   * @name MonitoringPostRunProbe
   * @summary Detailed Health Probe For the Instance
   * @request POST:/api/v1/monitoring/{tenant}/probe
   * @secure
   */
  monitoringPostRunProbe = (tenant: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/monitoring/${tenant}/probe`,
      method: "POST",
      secure: true,
      ...params,
    });
  /**
   * @description Get the version of the server
   *
   * @name InfoGetVersion
   * @summary We return the version for the currently running server
   * @request GET:/api/v1/version
   */
  infoGetVersion = (params: RequestParams = {}) =>
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
    });
}
