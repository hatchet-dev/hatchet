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
  APICloudMetadata,
  APIError,
  APIErrors,
  CreateManagedWorkerRequest,
  ListGithubAppInstallationsResponse,
  ListGithubBranchesResponse,
  ListGithubReposResponse,
  ManagedWorker,
  TenantBillingState,
  TenantSubscription,
  UpdateTenantSubscription,
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
}
