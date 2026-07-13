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
  AcceptOrganizationInviteRequest,
  APICloudMetadata,
  APIError,
  APIErrors,
  APITokenList,
  AuditLogList,
  Build,
  CreateManagedWorkerFromTemplateRequest,
  CreateManagedWorkerRequest,
  CreateManagementTokenRequest,
  CreateManagementTokenResponse,
  CreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest,
  CreateOrganizationRequest,
  CreateOrUpdateAutoscalingRequest,
  CreateTenantAPITokenRequest,
  CreateTenantAPITokenResponse,
  FeatureFlags,
  InfraAsCodeRequest,
  InstanceList,
  ListGithubAppInstallationsResponse,
  ListGithubBranchesResponse,
  ListGithubReposResponse,
  LogLineList,
  ManagedWorker,
  ManagedWorkerEventList,
  ManagedWorkerList,
  ManagementTokenList,
  Matrix,
  MonthlyComputeCost,
  Organization,
  OrganizationForUserList,
  OrganizationInviteList,
  OrganizationTenant,
  RedeemOfferRequest,
  RedeemOfferResponse,
  RejectOrganizationInviteRequest,
  RemoveOrganizationMembersRequest,
  RuntimeConfigActionsResponse,
  UpdateManagedWorkerRequest,
  UpdateOrganizationRequest,
  UpdateOrganizationTenantRequest,
  UserOffer,
  VectorPushRequest,
  WorkflowRunEventsMetricsCounts,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<
  SecurityDataType = unknown,
> extends HttpClient<SecurityDataType> {
  /**
   * @description Gets metadata for the Hatchet instance
   *
   * @tags Metadata
   * @name MetadataGet
   * @summary Get metadata
   * @request GET:/api/v1/cloud/metadata
   */
  metadataGet = Object.assign((params: RequestParams = {}) =>
    this.request<APICloudMetadata, APIErrors>({
      path: `/api/v1/cloud/metadata`,
      method: "GET",
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubAppOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/cloud/users/github-app/start
   * @secure
   */
  userUpdateGithubAppOauthStart = Object.assign((
    query?: {
      /** Redirect To */
      redirect_to?: string;
      /** With Repo Installation */
      with_repo_installation?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<any, void>({
      path: `/api/v1/cloud/users/github-app/start`,
      method: "GET",
      query: query,
      secure: true,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubAppOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/cloud/users/github-app/callback
   * @secure
   */
  userUpdateGithubAppOauthCallback = Object.assign((params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/cloud/users/github-app/callback`,
      method: "GET",
      secure: true,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Github App global webhook
   *
   * @tags Github
   * @name GithubUpdateGlobalWebhook
   * @summary Github app global webhook
   * @request POST:/api/v1/cloud/github/webhook
   */
  githubUpdateGlobalWebhook = Object.assign((params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/github/webhook`,
      method: "POST",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Github App tenant webhook
   *
   * @tags Github
   * @name GithubUpdateTenantWebhook
   * @summary Github app tenant webhook
   * @request POST:/api/v1/cloud/github/webhook/{webhook}
   */
  githubUpdateTenantWebhook = Object.assign((webhook: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/github/webhook/${webhook}`,
      method: "POST",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description List Github App installations
   *
   * @tags Github
   * @name GithubAppListInstallations
   * @summary List Github App installations
   * @request GET:/api/v1/cloud/github-app/installations
   * @secure
   */
  githubAppListInstallations = Object.assign((
    query?: {
      /**
       * The tenant id
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      tenant?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<ListGithubAppInstallationsResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description List Github App repositories
   *
   * @tags Github
   * @name GithubAppListRepos
   * @summary List Github App repositories
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/repos
   * @secure
   */
  githubAppListRepos = Object.assign((
    ghInstallation: string,
    query: {
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
    this.request<ListGithubReposResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations/${ghInstallation}/repos`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["gh-installation"],
    }), { resources: new Set<string>(["gh-installation"]) });
  /**
   * @description Link Github App installation to a tenant
   *
   * @tags Github
   * @name GithubAppUpdateInstallation
   * @summary Link Github App installation to a tenant
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/link
   * @secure
   */
  githubAppUpdateInstallation = Object.assign((
    ghInstallation: string,
    query: {
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
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/github-app/installations/${ghInstallation}/link`,
      method: "GET",
      query: query,
      secure: true,
      ...params,
      xResources: ["gh-installation"],
    }), { resources: new Set<string>(["gh-installation"]) });
  /**
   * @description List Github App branches
   *
   * @tags Github
   * @name GithubAppListBranches
   * @summary List Github App branches
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/repos/{gh-repo-owner}/{gh-repo-name}/branches
   * @secure
   */
  githubAppListBranches = Object.assign((
    ghInstallation: string,
    ghRepoOwner: string,
    ghRepoName: string,
    query: {
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
    this.request<ListGithubBranchesResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations/${ghInstallation}/repos/${ghRepoOwner}/${ghRepoName}/branches`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["gh-installation"],
    }), { resources: new Set<string>(["gh-installation"]) });
  /**
   * @description Get all managed workers for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerList
   * @summary List Managed Workers
   * @request GET:/api/v1/cloud/tenants/{tenant}/managed-worker
   * @secure
   */
  managedWorkerList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<ManagedWorkerList, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerCreate
   * @summary Create Managed Worker
   * @request POST:/api/v1/cloud/tenants/{tenant}/managed-worker
   * @secure
   */
  managedWorkerCreate = Object.assign((
    tenant: string,
    data: CreateManagedWorkerRequest,
    params: RequestParams = {},
  ) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a managed worker from a template
   *
   * @tags Managed Worker
   * @name ManagedWorkerTemplateCreate
   * @summary Create Managed Worker from Template
   * @request POST:/api/v1/cloud/tenants/{tenant}/managed-worker/template
   * @secure
   */
  managedWorkerTemplateCreate = Object.assign((
    tenant: string,
    data: CreateManagedWorkerFromTemplateRequest,
    params: RequestParams = {},
  ) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker/template`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get the total compute costs for the tenant
   *
   * @tags Cost
   * @name ComputeCostGet
   * @summary Get Managed Worker Cost
   * @request GET:/api/v1/cloud/tenants/{tenant}/managed-worker/cost
   * @secure
   */
  computeCostGet = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<MonthlyComputeCost, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker/cost`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerGet
   * @summary Get Managed Worker
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerGet = Object.assign((managedWorker: string, params: RequestParams = {}) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Update a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerUpdate
   * @summary Update Managed Worker
   * @request POST:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerUpdate = Object.assign((
    managedWorker: string,
    data: UpdateManagedWorkerRequest,
    params: RequestParams = {},
  ) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Delete a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerDelete
   * @summary Delete Managed Worker
   * @request DELETE:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerDelete = Object.assign((managedWorker: string, params: RequestParams = {}) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Registers runtime configs via infra-as-code
   *
   * @tags Managed Worker
   * @name InfraAsCodeCreate
   * @summary Create Infra as Code
   * @request POST:/api/v1/cloud/infra-as-code/{infra-as-code-request}
   * @secure
   */
  infraAsCodeCreate = Object.assign((
    infraAsCodeRequest: string,
    data: InfraAsCodeRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/infra-as-code/${infraAsCodeRequest}`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["tenant", "infra-as-code-request"],
    }), { resources: new Set<string>(["tenant", "infra-as-code-request"]) });
  /**
   * @description Get a list of runtime config actions for a managed worker
   *
   * @tags Managed Worker
   * @name RuntimeConfigListActions
   * @summary Get Runtime Config Actions
   * @request GET:/api/v1/cloud/runtime-config/{runtime-config}/actions
   * @secure
   */
  runtimeConfigListActions = Object.assign((
    runtimeConfig: string,
    params: RequestParams = {},
  ) =>
    this.request<RuntimeConfigActionsResponse, APIErrors>({
      path: `/api/v1/cloud/runtime-config/${runtimeConfig}/actions`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker", "runtime-config"],
    }), { resources: new Set<string>(["tenant", "managed-worker", "runtime-config"]) });
  /**
   * @description Get all instances for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerInstancesList
   * @summary List Instances
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/instances
   * @secure
   */
  managedWorkerInstancesList = Object.assign((
    managedWorker: string,
    params: RequestParams = {},
  ) =>
    this.request<InstanceList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/instances`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get a build
   *
   * @tags Build
   * @name BuildGet
   * @summary Get Build
   * @request GET:/api/v1/cloud/build/{build}
   * @secure
   */
  buildGet = Object.assign((build: string, params: RequestParams = {}) =>
    this.request<Build, APIErrors>({
      path: `/api/v1/cloud/build/${build}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker", "build"],
    }), { resources: new Set<string>(["tenant", "managed-worker", "build"]) });
  /**
   * @description Get events for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerEventsList
   * @summary Get Managed Worker Events
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/events
   * @secure
   */
  managedWorkerEventsList = Object.assign((
    managedWorker: string,
    params: RequestParams = {},
  ) =>
    this.request<ManagedWorkerEventList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/events`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get CPU metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsCpuGet
   * @summary Get CPU Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/cpu
   * @secure
   */
  metricsCpuGet = Object.assign((
    managedWorker: string,
    query?: {
      /**
       * When the metrics should start
       * @format date-time
       */
      after?: string;
      /**
       * When the metrics should end
       * @format date-time
       */
      before?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<Matrix, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/metrics/cpu`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get memory metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsMemoryGet
   * @summary Get Memory Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/memory
   * @secure
   */
  metricsMemoryGet = Object.assign((
    managedWorker: string,
    query?: {
      /**
       * When the metrics should start
       * @format date-time
       */
      after?: string;
      /**
       * When the metrics should end
       * @format date-time
       */
      before?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<Matrix, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/metrics/memory`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get disk metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsDiskGet
   * @summary Get Disk Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/disk
   * @secure
   */
  metricsDiskGet = Object.assign((
    managedWorker: string,
    query?: {
      /**
       * When the metrics should start
       * @format date-time
       */
      after?: string;
      /**
       * When the metrics should end
       * @format date-time
       */
      before?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<Matrix, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/metrics/disk`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get a minute by minute breakdown of workflow run metrics for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunEventsGetMetrics
   * @summary Get workflow runs
   * @request GET:/api/v1/cloud/tenants/{tenant}/runs-metrics
   * @secure
   */
  workflowRunEventsGetMetrics = Object.assign((
    tenant: string,
    query?: {
      /**
       * The time after the workflow run was created
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      createdAfter?: string;
      /**
       * The time before the workflow run was completed
       * @format date-time
       * @example "2021-01-01T00:00:00Z"
       */
      finishedBefore?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRunEventsMetricsCounts, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/runs-metrics`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Lists logs for a managed worker
   *
   * @tags Log
   * @name LogList
   * @summary List Logs
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/logs
   * @secure
   */
  logList = Object.assign((
    managedWorker: string,
    query?: {
      /**
       * When the logs should start
       * @format date-time
       */
      after?: string;
      /**
       * When the logs should end
       * @format date-time
       */
      before?: string;
      /** The search query to filter for */
      search?: string;
      /** The direction to sort the logs */
      direction?: "forward" | "backward";
    },
    params: RequestParams = {},
  ) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get the build logs for a specific build of a managed worker
   *
   * @tags Log
   * @name IacLogsList
   * @summary Get IaC Logs
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/iac-logs
   * @secure
   */
  iacLogsList = Object.assign((
    managedWorker: string,
    query: {
      /** The deploy key */
      deployKey: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/iac-logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker"],
    }), { resources: new Set<string>(["tenant", "managed-worker"]) });
  /**
   * @description Get the build logs for a specific build of a managed worker
   *
   * @tags Log
   * @name BuildLogsList
   * @summary Get Build Logs
   * @request GET:/api/v1/cloud/build/{build}/logs
   * @secure
   */
  buildLogsList = Object.assign((build: string, params: RequestParams = {}) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/cloud/build/${build}/logs`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "managed-worker", "build"],
    }), { resources: new Set<string>(["tenant", "managed-worker", "build"]) });
  /**
   * @description Push a log entry for the tenant
   *
   * @tags Log
   * @name LogCreate
   * @summary Push Log Entry
   * @request POST:/api/v1/cloud/tenants/{tenant}/logs
   * @secure
   */
  logCreate = Object.assign((
    tenant: string,
    data: VectorPushRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/logs`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description List all offers for the authenticated user
   *
   * @tags Billing
   * @name UserOffersList
   * @summary List offers for the authenticated user
   * @request GET:/api/v1/billing/offers
   * @secure
   */
  userOffersList = Object.assign((params: RequestParams = {}) =>
    this.request<UserOffer[], APIErrors>({
      path: `/api/v1/billing/offers`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Redeem an offer for the authenticated user, applying credit to the specified organization
   *
   * @tags Billing
   * @name UserOfferRedeem
   * @summary Redeem an offer
   * @request POST:/api/v1/billing/offers/redeem
   * @secure
   */
  userOfferRedeem = Object.assign((data: RedeemOfferRequest, params: RequestParams = {}) =>
    this.request<RedeemOfferResponse, APIErrors>({
      path: `/api/v1/billing/offers/redeem`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Get all feature flags for the tenant
   *
   * @tags Feature Flags
   * @name FeatureFlagsList
   * @summary List Feature Flags
   * @request GET:/api/v1/cloud/tenants/{tenant}/feature-flags
   * @secure
   */
  featureFlagsList = Object.assign((tenant: string, params: RequestParams = {}) =>
    this.request<FeatureFlags, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/feature-flags`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create autoscaling configuration for the tenant
   *
   * @tags Autoscaling Config
   * @name ExternalAutoscalingConfigCreate
   * @summary Create Autoscaling Config
   * @request POST:/api/v1/cloud/tenants/{tenant}/autoscaling
   * @secure
   */
  externalAutoscalingConfigCreate = Object.assign((
    tenant: string,
    data: CreateOrUpdateAutoscalingRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/autoscaling`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description List all organizations the authenticated user is a member of
   *
   * @name OrganizationList
   * @summary List Organizations
   * @request GET:/api/v1/management/organizations
   * @secure
   */
  organizationList = Object.assign((params: RequestParams = {}) =>
    this.request<OrganizationForUserList, APIError>({
      path: `/api/v1/management/organizations`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Create a new organization
   *
   * @name OrganizationCreate
   * @summary Create Organization
   * @request POST:/api/v1/management/organizations
   * @secure
   */
  organizationCreate = Object.assign((
    data: CreateOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<Organization, APIError>({
      path: `/api/v1/management/organizations`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Get organization details
   *
   * @tags Management
   * @name OrganizationGet
   * @summary Get Organization
   * @request GET:/api/v1/management/organizations/{organization}
   * @secure
   */
  organizationGet = Object.assign((organization: string, params: RequestParams = {}) =>
    this.request<Organization, APIError>({
      path: `/api/v1/management/organizations/${organization}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Update an organization
   *
   * @name OrganizationUpdate
   * @summary Update Organization
   * @request PATCH:/api/v1/management/organizations/{organization}
   * @secure
   */
  organizationUpdate = Object.assign((
    organization: string,
    data: UpdateOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<Organization, APIError>({
      path: `/api/v1/management/organizations/${organization}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Create a new tenant in the organization
   *
   * @tags Management
   * @name OrganizationCreateTenant
   * @summary Create Tenant in Organization
   * @request POST:/api/v1/management/organizations/{organization}/tenants
   * @secure
   */
  organizationCreateTenant = Object.assign((
    organization: string,
    data: CreateNewTenantForOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/management/organizations/${organization}/tenants`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Update a tenant in the organization
   *
   * @tags Management
   * @name OrganizationTenantUpdate
   * @summary Update Tenant in Organization
   * @request PATCH:/api/v1/management/organization-tenants/{organization-tenant}
   * @secure
   */
  organizationTenantUpdate = Object.assign((
    organizationTenant: string,
    data: UpdateOrganizationTenantRequest,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
  /**
   * @description Delete (archive) a tenant in the organization
   *
   * @tags Management
   * @name OrganizationTenantDelete
   * @summary Delete Tenant in Organization
   * @request DELETE:/api/v1/management/organization-tenants/{organization-tenant}
   * @secure
   */
  organizationTenantDelete = Object.assign((
    organizationTenant: string,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
  /**
   * @description List all API tokens for a tenant
   *
   * @tags Management
   * @name OrganizationTenantListApiTokens
   * @summary List API Tokens for Tenant
   * @request GET:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens
   * @secure
   */
  organizationTenantListApiTokens = Object.assign((
    organizationTenant: string,
    params: RequestParams = {},
  ) =>
    this.request<APITokenList, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}/api-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
  /**
   * @description Create a new API token for a tenant
   *
   * @tags Management
   * @name OrganizationTenantCreateApiToken
   * @summary Create API Token for Tenant
   * @request POST:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens
   * @secure
   */
  organizationTenantCreateApiToken = Object.assign((
    organizationTenant: string,
    data: CreateTenantAPITokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<CreateTenantAPITokenResponse, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}/api-tokens`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
  /**
   * @description Delete an API token for a tenant
   *
   * @tags Management
   * @name OrganizationTenantDeleteApiToken
   * @summary Delete API Token for Tenant
   * @request DELETE:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens/{api-token}
   * @secure
   */
  organizationTenantDeleteApiToken = Object.assign((
    organizationTenant: string,
    apiToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}/api-tokens/${apiToken}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
  /**
   * @description Remove a member from an organization
   *
   * @tags Management
   * @name OrganizationMemberDelete
   * @summary Remove Member from Organization
   * @request DELETE:/api/v1/management/organization-members/{organization-member}
   * @secure
   */
  organizationMemberDelete = Object.assign((
    organizationMember: string,
    data: RemoveOrganizationMembersRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organization-members/${organizationMember}`,
      method: "DELETE",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["organization", "organization-member"],
    }), { resources: new Set<string>(["organization", "organization-member"]) });
  /**
   * @description Create a new management token for an organization
   *
   * @tags Management
   * @name ManagementTokenCreate
   * @summary Create Management Token for Organization
   * @request POST:/api/v1/management/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenCreate = Object.assign((
    organization: string,
    data: CreateManagementTokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<CreateManagementTokenResponse, APIError>({
      path: `/api/v1/management/organizations/${organization}/management-tokens`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Get a management token for an organization
   *
   * @name ManagementTokenList
   * @summary Get Management Tokens for Organization
   * @request GET:/api/v1/management/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenList = Object.assign((organization: string, params: RequestParams = {}) =>
    this.request<ManagementTokenList, APIError>({
      path: `/api/v1/management/organizations/${organization}/management-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Delete a management token for an organization
   *
   * @name ManagementTokenDelete
   * @summary Delete Management Token for Organization
   * @request DELETE:/api/v1/management/management-tokens/{management-token}
   * @secure
   */
  managementTokenDelete = Object.assign((
    managementToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/management-tokens/${managementToken}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["organization", "management-token"],
    }), { resources: new Set<string>(["organization", "management-token"]) });
  /**
   * @description List all organization invites for the authenticated user
   *
   * @name UserListOrganizationInvites
   * @summary List Organization Invites for User
   * @request GET:/api/v1/management/invites
   * @secure
   */
  userListOrganizationInvites = Object.assign((params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/management/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Accept an organization invite
   *
   * @name OrganizationInviteAccept
   * @summary Accept Organization Invite for User
   * @request POST:/api/v1/management/invites/accept
   * @secure
   */
  organizationInviteAccept = Object.assign((
    data: AcceptOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/invites/accept`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description Reject an organization invite
   *
   * @name OrganizationInviteReject
   * @summary Reject Organization Invite for User
   * @request POST:/api/v1/management/invites/reject
   * @secure
   */
  organizationInviteReject = Object.assign((
    data: RejectOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/invites/reject`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: [],
    }), { resources: new Set<string>([]) });
  /**
   * @description List all organization invites for an organization
   *
   * @tags Management
   * @name OrganizationInviteList
   * @summary List Organization Invites for Organization
   * @request GET:/api/v1/management/organizations/{organization}/invites
   * @secure
   */
  organizationInviteList = Object.assign((organization: string, params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/management/organizations/${organization}/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Create a new organization invite
   *
   * @tags Management
   * @name OrganizationInviteCreate
   * @summary Create Organization Invite for Organization
   * @request POST:/api/v1/management/organizations/{organization}/invites
   * @secure
   */
  organizationInviteCreate = Object.assign((
    organization: string,
    data: CreateOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organizations/${organization}/invites`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
  /**
   * @description Delete an organization invite
   *
   * @tags Management
   * @name OrganizationInviteDelete
   * @summary Delete Organization Invite for Organization
   * @request DELETE:/api/v1/management/organization-invites/{organization-invite}
   * @secure
   */
  organizationInviteDelete = Object.assign((
    organizationInvite: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organization-invites/${organizationInvite}`,
      method: "DELETE",
      secure: true,
      ...params,
      xResources: ["organization", "organization-invite"],
    }), { resources: new Set<string>(["organization", "organization-invite"]) });
  /**
   * @description List all audit logs for an organization
   *
   * @tags Management
   * @name OrganizationListAuditLogs
   * @summary List Audit Logs for Organization
   * @request GET:/api/v1/management/organizations/{organization}/audit-logs
   * @secure
   */
  organizationListAuditLogs = Object.assign((
    organization: string,
    query?: {
      /**
       * The tenant ID belonging to the organization
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      tenant?: string;
      /**
       * The maximum number of audit logs to return
       * @format int32
       * @min 1
       * @max 1000
       * @default 1000
       */
      limit?: number;
      /**
       * The number of audit logs to skip
       * @format int32
       * @min 0
       * @default 0
       */
      offset?: number;
      /**
       * The start of the time range to filter audit logs (defaults to 24 hours ago)
       * @format date-time
       */
      since?: string;
      /**
       * The end of the time range to filter audit logs (defaults to now)
       * @format date-time
       */
      until?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<AuditLogList, APIError>({
      path: `/api/v1/management/organizations/${organization}/audit-logs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["organization"],
    }), { resources: new Set<string>(["organization"]) });
}
