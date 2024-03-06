/* eslint-disable */
/* tslint:disable */
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
  CreateAPITokenRequest,
  CreateAPITokenResponse,
  CreatePullRequestFromStepRun,
  CreateTenantInviteRequest,
  CreateTenantRequest,
  EventData,
  EventKey,
  EventKeyList,
  EventList,
  EventOrderByDirection,
  EventOrderByField,
  EventSearch,
  GetStepRunDiffResponse,
  LinkGithubRepositoryRequest,
  ListAPIMetaIntegration,
  ListAPITokensResponse,
  ListGithubAppInstallationsResponse,
  ListGithubBranchesResponse,
  ListGithubReposResponse,
  ListPullRequestsResponse,
  LogLineLevelField,
  LogLineList,
  LogLineOrderByDirection,
  LogLineOrderByField,
  LogLineSearch,
  PullRequestState,
  RejectInviteRequest,
  ReplayEventRequest,
  RerunStepRunRequest,
  StepRun,
  Tenant,
  TenantInvite,
  TenantInviteList,
  TenantMemberList,
  TriggerWorkflowRunRequest,
  UpdateTenantInviteRequest,
  User,
  UserLoginRequest,
  UserRegisterRequest,
  UserTenantMembershipsList,
  Worker,
  WorkerList,
  Workflow,
  WorkflowID,
  WorkflowList,
  WorkflowRun,
  WorkflowRunList,
  WorkflowRunStatusList,
  WorkflowVersion,
  WorkflowVersionDefinition,
} from './data-contracts';
import { ContentType, HttpClient, RequestParams } from './http-client';

export class Api<SecurityDataType = unknown> extends HttpClient<SecurityDataType> {
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
      method: 'GET',
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
      method: 'GET',
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
      method: 'GET',
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'POST',
      body: data,
      type: ContentType.Json,
      format: 'json',
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
      method: 'GET',
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
      method: 'GET',
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/users/github/start
   * @secure
   */
  userUpdateGithubOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/start`,
      method: 'GET',
      secure: true,
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/users/github/callback
   * @secure
   */
  userUpdateGithubOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/users/github/callback`,
      method: 'GET',
      secure: true,
      ...params,
    });
  /**
   * @description Github App global webhook
   *
   * @tags Github
   * @name GithubUpdateGlobalWebhook
   * @summary Github app global webhook
   * @request POST:/api/v1/github/webhook
   */
  githubUpdateGlobalWebhook = (params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/github/webhook`,
      method: 'POST',
      ...params,
    });
  /**
   * @description Github App tenant webhook
   *
   * @tags Github
   * @name GithubUpdateTenantWebhook
   * @summary Github app tenant webhook
   * @request POST:/api/v1/github/webhook/{webhook}
   */
  githubUpdateTenantWebhook = (webhook: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/github/webhook/${webhook}`,
      method: 'POST',
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
      method: 'POST',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'POST',
      body: data,
      type: ContentType.Json,
      format: 'json',
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
      method: 'POST',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
  tenantInviteAccept = (data: AcceptInviteRequest, params: RequestParams = {}) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/users/invites/accept`,
      method: 'POST',
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
  tenantInviteReject = (data: RejectInviteRequest, params: RequestParams = {}) =>
    this.request<void, APIErrors | APIError>({
      path: `/api/v1/users/invites/reject`,
      method: 'POST',
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
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
    params: RequestParams = {}
  ) =>
    this.request<TenantInvite, APIErrors | APIError>({
      path: `/api/v1/tenants/${tenant}/invites`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
    params: RequestParams = {}
  ) =>
    this.request<TenantInvite, APIErrors>({
      path: `/api/v1/tenants/${tenant}/invites/${tenantInvite}`,
      method: 'PATCH',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
  tenantInviteDelete = (tenant: string, tenantInvite: string, params: RequestParams = {}) =>
    this.request<TenantInvite, APIErrors>({
      path: `/api/v1/tenants/${tenant}/invites/${tenantInvite}`,
      method: 'DELETE',
      secure: true,
      format: 'json',
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
  apiTokenCreate = (tenant: string, data: CreateAPITokenRequest, params: RequestParams = {}) =>
    this.request<CreateAPITokenResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/api-tokens`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'POST',
      secure: true,
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
    },
    params: RequestParams = {}
  ) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
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
  eventUpdateReplay = (tenant: string, data: ReplayEventRequest, params: RequestParams = {}) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/replay`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
  workflowList = (tenant: string, params: RequestParams = {}) =>
    this.request<WorkflowList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows`,
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'DELETE',
      secure: true,
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
    params: RequestParams = {}
  ) =>
    this.request<WorkflowVersion, APIErrors>({
      path: `/api/v1/workflows/${workflow}/versions`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
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
    params: RequestParams = {}
  ) =>
    this.request<WorkflowRun, APIErrors>({
      path: `/api/v1/workflows/${workflow}/trigger`,
      method: 'POST',
      query: query,
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
      ...params,
    });
  /**
   * @description Get a workflow version definition for a tenant
   *
   * @tags Workflow
   * @name WorkflowVersionGetDefinition
   * @summary Get workflow version definition
   * @request GET:/api/v1/workflows/{workflow}/versions/definition
   * @secure
   */
  workflowVersionGetDefinition = (
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
    params: RequestParams = {}
  ) =>
    this.request<WorkflowVersionDefinition, APIErrors>({
      path: `/api/v1/workflows/${workflow}/versions/definition`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description Link a github repository to a workflow
   *
   * @tags Workflow
   * @name WorkflowUpdateLinkGithub
   * @summary Link github repository
   * @request POST:/api/v1/workflows/{workflow}/link-github
   * @secure
   */
  workflowUpdateLinkGithub = (
    workflow: string,
    data: LinkGithubRepositoryRequest,
    params: RequestParams = {}
  ) =>
    this.request<Workflow, APIErrors>({
      path: `/api/v1/workflows/${workflow}/link-github`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
      ...params,
    });
  /**
   * @description Create a pull request for a workflow
   *
   * @tags Workflow
   * @name StepRunUpdateCreatePr
   * @summary Create pull request
   * @request POST:/api/v1/step-runs/{step-run}/create-pr
   * @secure
   */
  stepRunUpdateCreatePr = (
    stepRun: string,
    data: CreatePullRequestFromStepRun,
    params: RequestParams = {}
  ) =>
    this.request<CreatePullRequestFromStepRun, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/create-pr`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
    params: RequestParams = {}
  ) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/logs`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description Get the diff for a step run between the most recent run and the first run.
   *
   * @tags Workflow
   * @name StepRunGetDiff
   * @summary Get diff
   * @request GET:/api/v1/step-runs/{step-run}/diff
   * @secure
   */
  stepRunGetDiff = (stepRun: string, params: RequestParams = {}) =>
    this.request<GetStepRunDiffResponse, APIErrors>({
      path: `/api/v1/step-runs/${stepRun}/diff`,
      method: 'GET',
      secure: true,
      format: 'json',
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
    },
    params: RequestParams = {}
  ) =>
    this.request<WorkflowRunList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/runs`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
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
  workflowRunGet = (tenant: string, workflowRun: string, params: RequestParams = {}) =>
    this.request<WorkflowRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}`,
      method: 'GET',
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description List all pull requests for a workflow run
   *
   * @tags Workflow
   * @name WorkflowRunListPullRequests
   * @summary List pull requests
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}/prs
   * @secure
   */
  workflowRunListPullRequests = (
    tenant: string,
    workflowRun: string,
    query?: {
      /** The pull request state */
      state?: PullRequestState;
    },
    params: RequestParams = {}
  ) =>
    this.request<ListPullRequestsResponse, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}/prs`,
      method: 'GET',
      query: query,
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
    params: RequestParams = {}
  ) =>
    this.request<StepRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}/rerun`,
      method: 'POST',
      body: data,
      secure: true,
      type: ContentType.Json,
      format: 'json',
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
  stepRunGetSchema = (tenant: string, stepRun: string, params: RequestParams = {}) =>
    this.request<object, APIErrors>({
      path: `/api/v1/tenants/${tenant}/step-runs/${stepRun}/schema`,
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
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
      method: 'GET',
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description List Github App installations
   *
   * @tags Github
   * @name GithubAppListInstallations
   * @summary List Github App installations
   * @request GET:/api/v1/github-app/installations
   * @secure
   */
  githubAppListInstallations = (params: RequestParams = {}) =>
    this.request<ListGithubAppInstallationsResponse, APIErrors>({
      path: `/api/v1/github-app/installations`,
      method: 'GET',
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description List Github App repositories
   *
   * @tags Github
   * @name GithubAppListRepos
   * @summary List Github App repositories
   * @request GET:/api/v1/github-app/installations/{gh-installation}/repos
   * @secure
   */
  githubAppListRepos = (ghInstallation: string, params: RequestParams = {}) =>
    this.request<ListGithubReposResponse, APIErrors>({
      path: `/api/v1/github-app/installations/${ghInstallation}/repos`,
      method: 'GET',
      secure: true,
      format: 'json',
      ...params,
    });
  /**
   * @description List Github App branches
   *
   * @tags Github
   * @name GithubAppListBranches
   * @summary List Github App branches
   * @request GET:/api/v1/github-app/installations/{gh-installation}/repos/{gh-repo-owner}/{gh-repo-name}/branches
   * @secure
   */
  githubAppListBranches = (
    ghInstallation: string,
    ghRepoOwner: string,
    ghRepoName: string,
    params: RequestParams = {}
  ) =>
    this.request<ListGithubBranchesResponse, APIErrors>({
      path: `/api/v1/github-app/installations/${ghInstallation}/repos/${ghRepoOwner}/${ghRepoName}/branches`,
      method: 'GET',
      secure: true,
      format: 'json',
      ...params,
    });
}
