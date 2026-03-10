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
  AutumnWebhookEvent,
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
  RejectOrganizationInviteRequest,
  RemoveOrganizationMembersRequest,
  RuntimeConfigActionsResponse,
  TenantBillingState,
  TenantCreditBalance,
  TenantPaymentMethodList,
  UpdateManagedWorkerRequest,
  UpdateOrganizationRequest,
  UpdateOrganizationTenantRequest,
  UpdateTenantSubscriptionRequest,
  UpdateTenantSubscriptionResponse,
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
  metadataGet = (params: RequestParams = {}) =>
    this.request<APICloudMetadata, APIErrors>({
      path: `/api/v1/cloud/metadata`,
      method: "GET",
      format: "json",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubAppOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/cloud/users/github-app/start
   * @secure
   */
  userUpdateGithubAppOauthStart = (
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
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name UserUpdateGithubAppOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/cloud/users/github-app/callback
   * @secure
   */
  userUpdateGithubAppOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/cloud/users/github-app/callback`,
      method: "GET",
      secure: true,
      ...params,
    });
  /**
   * @description Github App global webhook
   *
   * @tags Github
   * @name GithubUpdateGlobalWebhook
   * @summary Github app global webhook
   * @request POST:/api/v1/cloud/github/webhook
   */
  githubUpdateGlobalWebhook = (params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/github/webhook`,
      method: "POST",
      ...params,
    });
  /**
   * @description Github App tenant webhook
   *
   * @tags Github
   * @name GithubUpdateTenantWebhook
   * @summary Github app tenant webhook
   * @request POST:/api/v1/cloud/github/webhook/{webhook}
   */
  githubUpdateTenantWebhook = (webhook: string, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/github/webhook/${webhook}`,
      method: "POST",
      ...params,
    });
  /**
   * @description List Github App installations
   *
   * @tags Github
   * @name GithubAppListInstallations
   * @summary List Github App installations
   * @request GET:/api/v1/cloud/github-app/installations
   * @secure
   */
  githubAppListInstallations = (
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
    });
  /**
   * @description List Github App repositories
   *
   * @tags Github
   * @name GithubAppListRepos
   * @summary List Github App repositories
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/repos
   * @secure
   */
  githubAppListRepos = (
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
    });
  /**
   * @description Link Github App installation to a tenant
   *
   * @tags Github
   * @name GithubAppUpdateInstallation
   * @summary Link Github App installation to a tenant
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/link
   * @secure
   */
  githubAppUpdateInstallation = (
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
    });
  /**
   * @description List Github App branches
   *
   * @tags Github
   * @name GithubAppListBranches
   * @summary List Github App branches
   * @request GET:/api/v1/cloud/github-app/installations/{gh-installation}/repos/{gh-repo-owner}/{gh-repo-name}/branches
   * @secure
   */
  githubAppListBranches = (
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
    });
  /**
   * @description Get all managed workers for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerList
   * @summary List Managed Workers
   * @request GET:/api/v1/cloud/tenants/{tenant}/managed-worker
   * @secure
   */
  managedWorkerList = (tenant: string, params: RequestParams = {}) =>
    this.request<ManagedWorkerList, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerCreate
   * @summary Create Managed Worker
   * @request POST:/api/v1/cloud/tenants/{tenant}/managed-worker
   * @secure
   */
  managedWorkerCreate = (
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
    });
  /**
   * @description Create a managed worker from a template
   *
   * @tags Managed Worker
   * @name ManagedWorkerTemplateCreate
   * @summary Create Managed Worker from Template
   * @request POST:/api/v1/cloud/tenants/{tenant}/managed-worker/template
   * @secure
   */
  managedWorkerTemplateCreate = (
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
    });
  /**
   * @description Get the total compute costs for the tenant
   *
   * @tags Cost
   * @name ComputeCostGet
   * @summary Get Managed Worker Cost
   * @request GET:/api/v1/cloud/tenants/{tenant}/managed-worker/cost
   * @secure
   */
  computeCostGet = (tenant: string, params: RequestParams = {}) =>
    this.request<MonthlyComputeCost, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/managed-worker/cost`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerGet
   * @summary Get Managed Worker
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerGet = (managedWorker: string, params: RequestParams = {}) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerUpdate
   * @summary Update Managed Worker
   * @request POST:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerUpdate = (
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
    });
  /**
   * @description Delete a managed worker for the tenant
   *
   * @tags Managed Worker
   * @name ManagedWorkerDelete
   * @summary Delete Managed Worker
   * @request DELETE:/api/v1/cloud/managed-worker/{managed-worker}
   * @secure
   */
  managedWorkerDelete = (managedWorker: string, params: RequestParams = {}) =>
    this.request<ManagedWorker, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Registers runtime configs via infra-as-code
   *
   * @tags Managed Worker
   * @name InfraAsCodeCreate
   * @summary Create Infra as Code
   * @request POST:/api/v1/cloud/infra-as-code/{infra-as-code-request}
   * @secure
   */
  infraAsCodeCreate = (
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
    });
  /**
   * @description Get a list of runtime config actions for a managed worker
   *
   * @tags Managed Worker
   * @name RuntimeConfigListActions
   * @summary Get Runtime Config Actions
   * @request GET:/api/v1/cloud/runtime-config/{runtime-config}/actions
   * @secure
   */
  runtimeConfigListActions = (
    runtimeConfig: string,
    params: RequestParams = {},
  ) =>
    this.request<RuntimeConfigActionsResponse, APIErrors>({
      path: `/api/v1/cloud/runtime-config/${runtimeConfig}/actions`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all instances for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerInstancesList
   * @summary List Instances
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/instances
   * @secure
   */
  managedWorkerInstancesList = (
    managedWorker: string,
    params: RequestParams = {},
  ) =>
    this.request<InstanceList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/instances`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a build
   *
   * @tags Build
   * @name BuildGet
   * @summary Get Build
   * @request GET:/api/v1/cloud/build/{build}
   * @secure
   */
  buildGet = (build: string, params: RequestParams = {}) =>
    this.request<Build, APIErrors>({
      path: `/api/v1/cloud/build/${build}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get events for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerEventsList
   * @summary Get Managed Worker Events
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/events
   * @secure
   */
  managedWorkerEventsList = (
    managedWorker: string,
    params: RequestParams = {},
  ) =>
    this.request<ManagedWorkerEventList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/events`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get CPU metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsCpuGet
   * @summary Get CPU Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/cpu
   * @secure
   */
  metricsCpuGet = (
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
    });
  /**
   * @description Get memory metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsMemoryGet
   * @summary Get Memory Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/memory
   * @secure
   */
  metricsMemoryGet = (
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
    });
  /**
   * @description Get disk metrics for a managed worker
   *
   * @tags Metrics
   * @name MetricsDiskGet
   * @summary Get Disk Metrics
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/metrics/disk
   * @secure
   */
  metricsDiskGet = (
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
    });
  /**
   * @description Get a minute by minute breakdown of workflow run metrics for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunEventsGetMetrics
   * @summary Get workflow runs
   * @request GET:/api/v1/cloud/tenants/{tenant}/runs-metrics
   * @secure
   */
  workflowRunEventsGetMetrics = (
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
    });
  /**
   * @description Lists logs for a managed worker
   *
   * @tags Log
   * @name LogList
   * @summary List Logs
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/logs
   * @secure
   */
  logList = (
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
    });
  /**
   * @description Get the build logs for a specific build of a managed worker
   *
   * @tags Log
   * @name IacLogsList
   * @summary Get IaC Logs
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/iac-logs
   * @secure
   */
  iacLogsList = (
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
    });
  /**
   * @description Get the build logs for a specific build of a managed worker
   *
   * @tags Log
   * @name BuildLogsList
   * @summary Get Build Logs
   * @request GET:/api/v1/cloud/build/{build}/logs
   * @secure
   */
  buildLogsList = (build: string, params: RequestParams = {}) =>
    this.request<LogLineList, APIErrors>({
      path: `/api/v1/cloud/build/${build}/logs`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Push a log entry for the tenant
   *
   * @tags Log
   * @name LogCreate
   * @summary Push Log Entry
   * @request POST:/api/v1/cloud/tenants/{tenant}/logs
   * @secure
   */
  logCreate = (
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
    });
  /**
   * @description Receive a webhook message from Autumn
   *
   * @tags Billing
   * @name AutumnEventCreate
   * @summary Receive a webhook message from Autumn
   * @request POST:/api/v1/billing/autumn/webhook
   */
  autumnEventCreate = (data: AutumnWebhookEvent, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/billing/autumn/webhook`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Gets the billing state for a tenant
   *
   * @tags Tenant
   * @name TenantBillingStateGet
   * @summary Get the billing state for a tenant
   * @request GET:/api/v1/billing/tenants/{tenant}
   * @secure
   */
  tenantBillingStateGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantBillingState, APIErrors | APIError>({
      path: `/api/v1/billing/tenants/${tenant}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a subscription
   *
   * @tags Billing
   * @name TenantSubscriptionUpdate
   * @summary Create a new subscription
   * @request PATCH:/api/v1/billing/tenants/{tenant}/subscription
   * @secure
   */
  tenantSubscriptionUpdate = (
    tenant: string,
    data: UpdateTenantSubscriptionRequest,
    params: RequestParams = {},
  ) =>
    this.request<UpdateTenantSubscriptionResponse, APIErrors>({
      path: `/api/v1/billing/tenants/${tenant}/subscription`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Get the billing portal link
   *
   * @tags Billing
   * @name BillingPortalLinkGet
   * @summary Create a link to the billing portal
   * @request GET:/api/v1/billing/tenants/{tenant}/billing-portal-link
   * @secure
   */
  billingPortalLinkGet = (tenant: string, params: RequestParams = {}) =>
    this.request<
      {
        /** The url to the billing portal */
        url?: string;
      },
      APIErrors
    >({
      path: `/api/v1/billing/tenants/${tenant}/billing-portal-link`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get the payment methods for a tenant
   *
   * @tags Billing
   * @name TenantPaymentMethodsGet
   * @summary Get the payment methods for a tenant
   * @request GET:/api/v1/billing/tenants/{tenant}/payment-methods
   * @secure
   */
  tenantPaymentMethodsGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantPaymentMethodList, APIErrors>({
      path: `/api/v1/billing/tenants/${tenant}/payment-methods`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get the Stripe credit balance for a tenant
   *
   * @tags Billing
   * @name TenantCreditBalanceGet
   * @summary Get the Stripe credit balance for a tenant
   * @request GET:/api/v1/billing/tenants/{tenant}/credit-balance
   * @secure
   */
  tenantCreditBalanceGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantCreditBalance, APIErrors>({
      path: `/api/v1/billing/tenants/${tenant}/credit-balance`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all feature flags for the tenant
   *
   * @tags Feature Flags
   * @name FeatureFlagsList
   * @summary List Feature Flags
   * @request GET:/api/v1/cloud/tenants/{tenant}/feature-flags
   * @secure
   */
  featureFlagsList = (tenant: string, params: RequestParams = {}) =>
    this.request<FeatureFlags, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/feature-flags`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create autoscaling configuration for the tenant
   *
   * @tags Autoscaling Config
   * @name ExternalAutoscalingConfigCreate
   * @summary Create Autoscaling Config
   * @request POST:/api/v1/cloud/tenants/{tenant}/autoscaling
   * @secure
   */
  externalAutoscalingConfigCreate = (
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
    });
  /**
   * @description List all organizations the authenticated user is a member of
   *
   * @name OrganizationList
   * @summary List Organizations
   * @request GET:/api/v1/management/organizations
   * @secure
   */
  organizationList = (params: RequestParams = {}) =>
    this.request<OrganizationForUserList, APIError>({
      path: `/api/v1/management/organizations`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new organization
   *
   * @name OrganizationCreate
   * @summary Create Organization
   * @request POST:/api/v1/management/organizations
   * @secure
   */
  organizationCreate = (
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
    });
  /**
   * @description Get organization details
   *
   * @tags Management
   * @name OrganizationGet
   * @summary Get Organization
   * @request GET:/api/v1/management/organizations/{organization}
   * @secure
   */
  organizationGet = (organization: string, params: RequestParams = {}) =>
    this.request<Organization, APIError>({
      path: `/api/v1/management/organizations/${organization}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update an organization
   *
   * @name OrganizationUpdate
   * @summary Update Organization
   * @request PATCH:/api/v1/management/organizations/{organization}
   * @secure
   */
  organizationUpdate = (
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
    });
  /**
   * @description Create a new tenant in the organization
   *
   * @tags Management
   * @name OrganizationCreateTenant
   * @summary Create Tenant in Organization
   * @request POST:/api/v1/management/organizations/{organization}/tenants
   * @secure
   */
  organizationCreateTenant = (
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
    });
  /**
   * @description Update a tenant in the organization
   *
   * @tags Management
   * @name OrganizationTenantUpdate
   * @summary Update Tenant in Organization
   * @request PATCH:/api/v1/management/organization-tenants/{organization-tenant}
   * @secure
   */
  organizationTenantUpdate = (
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
    });
  /**
   * @description Delete (archive) a tenant in the organization
   *
   * @tags Management
   * @name OrganizationTenantDelete
   * @summary Delete Tenant in Organization
   * @request DELETE:/api/v1/management/organization-tenants/{organization-tenant}
   * @secure
   */
  organizationTenantDelete = (
    organizationTenant: string,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description List all API tokens for a tenant
   *
   * @tags Management
   * @name OrganizationTenantListApiTokens
   * @summary List API Tokens for Tenant
   * @request GET:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens
   * @secure
   */
  organizationTenantListApiTokens = (
    organizationTenant: string,
    params: RequestParams = {},
  ) =>
    this.request<APITokenList, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}/api-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new API token for a tenant
   *
   * @tags Management
   * @name OrganizationTenantCreateApiToken
   * @summary Create API Token for Tenant
   * @request POST:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens
   * @secure
   */
  organizationTenantCreateApiToken = (
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
    });
  /**
   * @description Delete an API token for a tenant
   *
   * @tags Management
   * @name OrganizationTenantDeleteApiToken
   * @summary Delete API Token for Tenant
   * @request DELETE:/api/v1/management/organization-tenants/{organization-tenant}/api-tokens/{api-token}
   * @secure
   */
  organizationTenantDeleteApiToken = (
    organizationTenant: string,
    apiToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organization-tenants/${organizationTenant}/api-tokens/${apiToken}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Remove a member from an organization
   *
   * @tags Management
   * @name OrganizationMemberDelete
   * @summary Remove Member from Organization
   * @request DELETE:/api/v1/management/organization-members/{organization-member}
   * @secure
   */
  organizationMemberDelete = (
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
    });
  /**
   * @description Create a new management token for an organization
   *
   * @tags Management
   * @name ManagementTokenCreate
   * @summary Create Management Token for Organization
   * @request POST:/api/v1/management/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenCreate = (
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
    });
  /**
   * @description Get a management token for an organization
   *
   * @name ManagementTokenList
   * @summary Get Management Tokens for Organization
   * @request GET:/api/v1/management/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenList = (organization: string, params: RequestParams = {}) =>
    this.request<ManagementTokenList, APIError>({
      path: `/api/v1/management/organizations/${organization}/management-tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Delete a management token for an organization
   *
   * @name ManagementTokenDelete
   * @summary Delete Management Token for Organization
   * @request DELETE:/api/v1/management/management-tokens/{management-token}
   * @secure
   */
  managementTokenDelete = (
    managementToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/management-tokens/${managementToken}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all organization invites for the authenticated user
   *
   * @name UserListOrganizationInvites
   * @summary List Organization Invites for User
   * @request GET:/api/v1/management/invites
   * @secure
   */
  userListOrganizationInvites = (params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/management/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Accept an organization invite
   *
   * @name OrganizationInviteAccept
   * @summary Accept Organization Invite for User
   * @request POST:/api/v1/management/invites/accept
   * @secure
   */
  organizationInviteAccept = (
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
    });
  /**
   * @description Reject an organization invite
   *
   * @name OrganizationInviteReject
   * @summary Reject Organization Invite for User
   * @request POST:/api/v1/management/invites/reject
   * @secure
   */
  organizationInviteReject = (
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
    });
  /**
   * @description List all organization invites for an organization
   *
   * @tags Management
   * @name OrganizationInviteList
   * @summary List Organization Invites for Organization
   * @request GET:/api/v1/management/organizations/{organization}/invites
   * @secure
   */
  organizationInviteList = (organization: string, params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/management/organizations/${organization}/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new organization invite
   *
   * @tags Management
   * @name OrganizationInviteCreate
   * @summary Create Organization Invite for Organization
   * @request POST:/api/v1/management/organizations/{organization}/invites
   * @secure
   */
  organizationInviteCreate = (
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
    });
  /**
   * @description Delete an organization invite
   *
   * @tags Management
   * @name OrganizationInviteDelete
   * @summary Delete Organization Invite for Organization
   * @request DELETE:/api/v1/management/organization-invites/{organization-invite}
   * @secure
   */
  organizationInviteDelete = (
    organizationInvite: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organization-invites/${organizationInvite}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
}
