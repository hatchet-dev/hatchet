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
  APICloudMetadata,
  APIError,
  APIErrors,
  Build,
  CreateManagedWorkerRequest,
  FeatureFlags,
  InstanceList,
  ListGithubAppInstallationsResponse,
  ListGithubBranchesResponse,
  ListGithubReposResponse,
  LogLineList,
  ManagedWorker,
  ManagedWorkerEventList,
  ManagedWorkerList,
  Matrix,
  TenantBillingState,
  TenantSubscription,
  UpdateTenantSubscription,
  VectorPushRequest,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<SecurityDataType = unknown> extends HttpClient<SecurityDataType> {
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
  userUpdateGithubAppOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/cloud/users/github-app/start`,
      method: "GET",
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
  githubAppListInstallations = (params: RequestParams = {}) =>
    this.request<ListGithubAppInstallationsResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations`,
      method: "GET",
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
  githubAppListRepos = (ghInstallation: string, params: RequestParams = {}) =>
    this.request<ListGithubReposResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations/${ghInstallation}/repos`,
      method: "GET",
      secure: true,
      format: "json",
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
    params: RequestParams = {},
  ) =>
    this.request<ListGithubBranchesResponse, APIErrors>({
      path: `/api/v1/cloud/github-app/installations/${ghInstallation}/repos/${ghRepoOwner}/${ghRepoName}/branches`,
      method: "GET",
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
  managedWorkerCreate = (tenant: string, data: CreateManagedWorkerRequest, params: RequestParams = {}) =>
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
  managedWorkerUpdate = (managedWorker: string, data: CreateManagedWorkerRequest, params: RequestParams = {}) =>
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
   * @description Get all instances for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerInstancesList
   * @summary List Instances
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/instances
   * @secure
   */
  managedWorkerInstancesList = (managedWorker: string, params: RequestParams = {}) =>
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
   * @description Get events for a managed worker
   *
   * @tags Managed Worker
   * @name ManagedWorkerEventsList
   * @summary Get Managed Worker Events
   * @request GET:/api/v1/cloud/managed-worker/{managed-worker}/events
   * @secure
   */
  managedWorkerEventsList = (managedWorker: string, params: RequestParams = {}) =>
    this.request<ManagedWorkerEventList, APIErrors>({
      path: `/api/v1/cloud/managed-worker/${managedWorker}/events`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Receive a webhook message from Lago
   *
   * @tags Billing
   * @name LagoMessageCreate
   * @summary Receive a webhook message from Lago
   * @request POST:/api/v1/billing/lago/webhook
   */
  lagoMessageCreate = (params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/billing/lago/webhook`,
      method: "POST",
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
   * @name SubscriptionUpsert
   * @summary Create a new subscription
   * @request PATCH:/api/v1/billing/tenants/{tenant}/subscription
   * @secure
   */
  subscriptionUpsert = (tenant: string, data: UpdateTenantSubscription, params: RequestParams = {}) =>
    this.request<TenantSubscription, APIErrors>({
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
   * @description Push a log entry for the tenant
   *
   * @tags Log
   * @name LogCreate
   * @summary Push Log Entry
   * @request POST:/api/v1/cloud/tenants/{tenant}/logs
   * @secure
   */
  logCreate = (tenant: string, data: VectorPushRequest, params: RequestParams = {}) =>
    this.request<void, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/logs`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
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
}
