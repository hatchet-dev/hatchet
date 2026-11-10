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
  AddOrganizationMembersToTenantRequest,
  APICloudMetadata,
  APIError,
  APIErrors,
  APITokenList,
  CreateManagementTokenRequest,
  CreateManagementTokenResponse,
  CreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest,
  CreateOrganizationRequest,
  CreateSandboxProviderRequest,
  CreateTenantAPITokenRequest,
  CreateTenantAPITokenResponse,
  ManagementTokenList,
  Organization,
  OrganizationForUserList,
  OrganizationInviteList,
  OrganizationTenant,
  RedeemOfferRequest,
  RedeemOfferResponse,
  RejectOrganizationInviteRequest,
  RemoveOrganizationMembersRequest,
  SandboxProvider,
  SandboxProviderEventList,
  SandboxProviderList,
  UpdateOrganizationRequest,
  UpdateOrganizationTenantRequest,
  UpdateSandboxProviderRequest,
  UserOffer,
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
   * @description List the sandbox providers of the tenant
   *
   * @tags Sandbox Provider
   * @name SandboxProviderList
   * @summary List Sandbox Providers
   * @request GET:/api/v1/cloud/tenants/{tenant}/sandbox-providers
   * @secure
   */
  sandboxProviderList = Object.assign((
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
    },
    params: RequestParams = {},
  ) =>
    this.request<SandboxProviderList, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/sandbox-providers`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Create a sandbox provider for the tenant. Requires the sandbox operator entitlement.
   *
   * @tags Sandbox Provider
   * @name SandboxProviderCreate
   * @summary Create Sandbox Provider
   * @request POST:/api/v1/cloud/tenants/{tenant}/sandbox-providers
   * @secure
   */
  sandboxProviderCreate = Object.assign((
    tenant: string,
    data: CreateSandboxProviderRequest,
    params: RequestParams = {},
  ) =>
    this.request<SandboxProvider, APIErrors>({
      path: `/api/v1/cloud/tenants/${tenant}/sandbox-providers`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant"],
    }), { resources: new Set<string>(["tenant"]) });
  /**
   * @description Get a sandbox provider
   *
   * @tags Sandbox Provider
   * @name SandboxProviderGet
   * @summary Get Sandbox Provider
   * @request GET:/api/v1/cloud/sandbox-providers/{sandbox-provider}
   * @secure
   */
  sandboxProviderGet = Object.assign((sandboxProvider: string, params: RequestParams = {}) =>
    this.request<SandboxProvider, APIErrors>({
      path: `/api/v1/cloud/sandbox-providers/${sandboxProvider}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "sandbox-provider"],
    }), { resources: new Set<string>(["tenant", "sandbox-provider"]) });
  /**
   * @description Update a sandbox provider
   *
   * @tags Sandbox Provider
   * @name SandboxProviderUpdate
   * @summary Update Sandbox Provider
   * @request PATCH:/api/v1/cloud/sandbox-providers/{sandbox-provider}
   * @secure
   */
  sandboxProviderUpdate = Object.assign((
    sandboxProvider: string,
    data: UpdateSandboxProviderRequest,
    params: RequestParams = {},
  ) =>
    this.request<SandboxProvider, APIErrors>({
      path: `/api/v1/cloud/sandbox-providers/${sandboxProvider}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
      xResources: ["tenant", "sandbox-provider"],
    }), { resources: new Set<string>(["tenant", "sandbox-provider"]) });
  /**
   * @description Delete a sandbox provider. Its sandboxes can no longer be reached.
   *
   * @tags Sandbox Provider
   * @name SandboxProviderDelete
   * @summary Delete Sandbox Provider
   * @request DELETE:/api/v1/cloud/sandbox-providers/{sandbox-provider}
   * @secure
   */
  sandboxProviderDelete = Object.assign((
    sandboxProvider: string,
    params: RequestParams = {},
  ) =>
    this.request<SandboxProvider, APIErrors>({
      path: `/api/v1/cloud/sandbox-providers/${sandboxProvider}`,
      method: "DELETE",
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "sandbox-provider"],
    }), { resources: new Set<string>(["tenant", "sandbox-provider"]) });
  /**
   * @description List what the sandbox operator observed about the provider, newest first, within the tenant's retention window.
   *
   * @tags Sandbox Provider
   * @name SandboxProviderEventsList
   * @summary List Sandbox Provider Events
   * @request GET:/api/v1/cloud/sandbox-providers/{sandbox-provider}/events
   * @secure
   */
  sandboxProviderEventsList = Object.assign((
    sandboxProvider: string,
    query?: {
      /**
       * The number of events to return, at most 500. Defaults to 100.
       * @format int64
       */
      limit?: number;
      /**
       * Only return events created after this time
       * @format date-time
       */
      since?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<SandboxProviderEventList, APIErrors>({
      path: `/api/v1/cloud/sandbox-providers/${sandboxProvider}/events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
      xResources: ["tenant", "sandbox-provider"],
    }), { resources: new Set<string>(["tenant", "sandbox-provider"]) });
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
   * @description Directly add one or more org members to a tenant, bypassing tag matching. Only org OWNERs can call this endpoint.
   *
   * @tags Management
   * @name OrganizationTenantMembersAdd
   * @summary Add Org Members to Tenant
   * @request POST:/api/v1/management/organizations/{organization}/tenants/{tenant}/members
   * @secure
   */
  organizationTenantMembersAdd = Object.assign((
    organization: string,
    tenant: string,
    data: AddOrganizationMembersToTenantRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/management/organizations/${organization}/tenants/${tenant}/members`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
      xResources: ["organization", "organization-tenant"],
    }), { resources: new Set<string>(["organization", "organization-tenant"]) });
}
