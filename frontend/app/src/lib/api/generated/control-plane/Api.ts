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
  AcceptTenantInviteRequest,
  APIControlPlaneMetadata,
  APIError,
  APIErrors,
  APIMetaAuth,
  APITokenList,
  CreateManagementTokenRequest,
  CreateManagementTokenResponse,
  CreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest,
  CreateOrganizationRequest,
  CreateOrganizationSsoDomainRequest,
  CreateTenantAPITokenRequest,
  CreateTenantAPITokenResponse,
  CreateTenantInviteRequest,
  ListAPIMetaIntegration,
  ManagementTokenList,
  Organization,
  OrganizationAvailableShardList,
  OrganizationForUserList,
  OrganizationInviteList,
  OrganizationTenant,
  RejectOrganizationInviteRequest,
  RejectTenantInviteRequest,
  RemoveOrganizationMembersRequest,
  SsoConfig,
  SsoDomainArray,
  SubscriptionPlanList,
  TenantBillingState,
  TenantCreditBalance,
  TenantExchangeToken,
  TenantInvite,
  TenantInviteList,
  TenantMember,
  TenantMemberList,
  TenantPaymentMethodList,
  UpdateOrganizationRequest,
  UpdateTenantInviteRequest,
  UpdateTenantMemberRequest,
  UpdateTenantSubscriptionRequest,
  UpdateTenantSubscriptionResponse,
  User,
  UserChangePasswordRequest,
  UserLoginRequest,
  UserRegisterRequest,
  UserTenantMembershipsList,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<
  SecurityDataType = unknown,
> extends HttpClient<SecurityDataType> {
  /**
   * @description Gets the readiness status
   *
   * @tags Healthcheck
   * @name ReadinessGet
   * @summary Get readiness
   * @request GET:/api/ready
   */
  readinessGet = (params: RequestParams = {}) =>
    this.request<void, APIErrors>({
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
    this.request<void, APIErrors>({
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
   * @request GET:/api/v1/control-plane/metadata
   */
  metadataGet = (params: RequestParams = {}) =>
    this.request<APIControlPlaneMetadata, APIErrors>({
      path: `/api/v1/control-plane/metadata`,
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
   * @request GET:/api/v1/control-plane/metadata/integrations
   * @secure
   */
  metadataListIntegrations = (params: RequestParams = {}) =>
    this.request<ListAPIMetaIntegration, APIErrors>({
      path: `/api/v1/control-plane/metadata/integrations`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Logs in a cloud user.
   *
   * @tags User
   * @name CloudUserUpdateLogin
   * @summary Login user
   * @request POST:/api/v1/control-plane/users/login
   */
  cloudUserUpdateLogin = (data: UserLoginRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/control-plane/users/login`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Logs out a cloud user.
   *
   * @tags User
   * @name CloudUserUpdateLogout
   * @summary Logout user
   * @request POST:/api/v1/control-plane/users/logout
   * @secure
   */
  cloudUserUpdateLogout = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/control-plane/users/logout`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a cloud user password.
   *
   * @tags User
   * @name CloudUserUpdatePassword
   * @summary Change user password
   * @request POST:/api/v1/control-plane/users/password
   * @secure
   */
  cloudUserUpdatePassword = (
    data: UserChangePasswordRequest,
    params: RequestParams = {},
  ) =>
    this.request<User, APIErrors>({
      path: `/api/v1/control-plane/users/password`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Registers a cloud user.
   *
   * @tags User
   * @name CloudUserCreate
   * @summary Register user
   * @request POST:/api/v1/control-plane/users/register
   */
  cloudUserCreate = (data: UserRegisterRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/control-plane/users/register`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Gets the current cloud user
   *
   * @tags User
   * @name CloudUserGetCurrent
   * @summary Get current cloud user
   * @request GET:/api/v1/control-plane/users/current
   * @secure
   */
  cloudUserGetCurrent = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/control-plane/users/current`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateGoogleOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/control-plane/users/google/start
   */
  cloudUserUpdateGoogleOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/google/start`,
      method: "GET",
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateGoogleOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/control-plane/users/google/callback
   */
  cloudUserUpdateGoogleOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/google/callback`,
      method: "GET",
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateGithubOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/control-plane/users/github/start
   */
  cloudUserUpdateGithubOauthStart = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/github/start`,
      method: "GET",
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateGithubOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/control-plane/users/github/callback
   */
  cloudUserUpdateGithubOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/github/callback`,
      method: "GET",
      ...params,
    });
  /**
   * @description Starts the Slack OAuth flow for a tenant
   *
   * @tags User
   * @name CloudUserUpdateSlackOauthStart
   * @summary Start Slack OAuth flow
   * @request GET:/api/v1/control-plane/tenants/{tenant}/slack/start
   * @secure
   */
  cloudUserUpdateSlackOauthStart = (
    tenant: string,
    params: RequestParams = {},
  ) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/tenants/${tenant}/slack/start`,
      method: "GET",
      secure: true,
      ...params,
    });
  /**
   * @description Completes the Slack OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateSlackOauthCallback
   * @summary Complete Slack OAuth flow
   * @request GET:/api/v1/control-plane/users/slack/callback
   * @secure
   */
  cloudUserUpdateSlackOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/slack/callback`,
      method: "GET",
      secure: true,
      ...params,
    });
  /**
   * @description List Hatchet deployment shards in the SHARED pool (available to any organization without dedicated shards).
   *
   * @tags Management
   * @name ShardsListShared
   * @summary List SHARED deployment shards
   * @request GET:/api/v1/control-plane/shared-shards
   * @secure
   */
  shardsListShared = (params: RequestParams = {}) =>
    this.request<OrganizationAvailableShardList, APIError>({
      path: `/api/v1/control-plane/shared-shards`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description List all organizations the authenticated user is a member of
   *
   * @name OrganizationList
   * @summary List Organizations
   * @request GET:/api/v1/control-plane/organizations
   * @secure
   */
  organizationList = (params: RequestParams = {}) =>
    this.request<OrganizationForUserList, APIError>({
      path: `/api/v1/control-plane/organizations`,
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
   * @request POST:/api/v1/control-plane/organizations
   * @secure
   */
  organizationCreate = (
    data: CreateOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<Organization, APIError>({
      path: `/api/v1/control-plane/organizations`,
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
   * @request GET:/api/v1/control-plane/organizations/{organization}
   * @secure
   */
  organizationGet = (organization: string, params: RequestParams = {}) =>
    this.request<Organization, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}`,
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
   * @request PATCH:/api/v1/control-plane/organizations/{organization}
   * @secure
   */
  organizationUpdate = (
    organization: string,
    data: UpdateOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<Organization, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}`,
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
   * @request POST:/api/v1/control-plane/organizations/{organization}/tenants
   * @secure
   */
  organizationCreateTenant = (
    organization: string,
    data: CreateNewTenantForOrganizationRequest,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/tenants`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description List Hatchet deployment shards available for new tenants in this organization
   *
   * @tags Management
   * @name OrganizationListAvailableShards
   * @summary List available deployment shards for organization
   * @request GET:/api/v1/control-plane/organizations/{organization}/available-shards
   * @secure
   */
  organizationListAvailableShards = (
    organization: string,
    params: RequestParams = {},
  ) =>
    this.request<OrganizationAvailableShardList, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/available-shards`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Delete (archive) a tenant in the organization
   *
   * @tags Management
   * @name OrganizationTenantDelete
   * @summary Delete Tenant in Organization
   * @request DELETE:/api/v1/control-plane/organization-tenants/{tenant}
   * @secure
   */
  organizationTenantDelete = (tenant: string, params: RequestParams = {}) =>
    this.request<OrganizationTenant, APIError>({
      path: `/api/v1/control-plane/organization-tenants/${tenant}`,
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
   * @request GET:/api/v1/control-plane/organization-tenants/{tenant}/api-tokens
   * @secure
   */
  organizationTenantListApiTokens = (
    tenant: string,
    params: RequestParams = {},
  ) =>
    this.request<APITokenList, APIError>({
      path: `/api/v1/control-plane/organization-tenants/${tenant}/api-tokens`,
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
   * @request POST:/api/v1/control-plane/organization-tenants/{tenant}/api-tokens
   * @secure
   */
  organizationTenantCreateApiToken = (
    tenant: string,
    data: CreateTenantAPITokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<CreateTenantAPITokenResponse, APIError>({
      path: `/api/v1/control-plane/organization-tenants/${tenant}/api-tokens`,
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
   * @request DELETE:/api/v1/control-plane/organization-tenants/{tenant}/api-tokens/{api-token}
   * @secure
   */
  organizationTenantDeleteApiToken = (
    tenant: string,
    apiToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organization-tenants/${tenant}/api-tokens/${apiToken}`,
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
   * @request DELETE:/api/v1/control-plane/organization-members/{organization-member}
   * @secure
   */
  organizationMemberDelete = (
    organizationMember: string,
    data: RemoveOrganizationMembersRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organization-members/${organizationMember}`,
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
   * @request POST:/api/v1/control-plane/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenCreate = (
    organization: string,
    data: CreateManagementTokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<CreateManagementTokenResponse, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/management-tokens`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Get management tokens for an organization
   *
   * @name ManagementTokenList
   * @summary Get Management Tokens for Organization
   * @request GET:/api/v1/control-plane/organizations/{organization}/management-tokens
   * @secure
   */
  managementTokenList = (organization: string, params: RequestParams = {}) =>
    this.request<ManagementTokenList, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/management-tokens`,
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
   * @request DELETE:/api/v1/control-plane/management-tokens/{management-token}
   * @secure
   */
  managementTokenDelete = (
    managementToken: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/management-tokens/${managementToken}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all organization invites for the authenticated user
   *
   * @name UserListOrganizationInvites
   * @summary List Organization Invites for User
   * @request GET:/api/v1/control-plane/invites
   * @secure
   */
  userListOrganizationInvites = (params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/control-plane/invites`,
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
   * @request POST:/api/v1/control-plane/invites/accept
   * @secure
   */
  organizationInviteAccept = (
    data: AcceptOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/invites/accept`,
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
   * @request POST:/api/v1/control-plane/invites/reject
   * @secure
   */
  organizationInviteReject = (
    data: RejectOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/invites/reject`,
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
   * @request GET:/api/v1/control-plane/organizations/{organization}/invites
   * @secure
   */
  organizationInviteList = (organization: string, params: RequestParams = {}) =>
    this.request<OrganizationInviteList, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/invites`,
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
   * @request POST:/api/v1/control-plane/organizations/{organization}/invites
   * @secure
   */
  organizationInviteCreate = (
    organization: string,
    data: CreateOrganizationInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/invites`,
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
   * @request DELETE:/api/v1/control-plane/organization-invites/{organization-invite}
   * @secure
   */
  organizationInviteDelete = (
    organizationInvite: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organization-invites/${organizationInvite}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all SSO configurations the organization has created
   *
   * @name SsoList
   * @summary List Organization's SSO Configs
   * @request GET:/api/v1/control-plane/organizations/{organization}/sso
   * @secure
   */
  ssoList = (organization: string, params: RequestParams = {}) =>
    this.request<APIMetaAuth, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new organization SSO config
   *
   * @name SsoUpdate
   * @summary Upsert organization SSO config
   * @request POST:/api/v1/control-plane/organizations/{organization}/sso
   * @secure
   */
  ssoUpdate = (
    organization: string,
    data: {
      idpInfoFromCustomer: object;
    },
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Delete organization SSO config
   *
   * @name SsoDelete
   * @request DELETE:/api/v1/control-plane/organizations/{organization}/sso
   * @secure
   */
  ssoDelete = (organization: string, params: RequestParams = {}) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all SSO domains for organization
   *
   * @name SsoDomainList
   * @summary List Organization's SSO Domains
   * @request GET:/api/v1/control-plane/organizations/{organization}/sso-domain
   * @secure
   */
  ssoDomainList = (organization: string, params: RequestParams = {}) =>
    this.request<SsoDomainArray, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso-domain`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Add a new SSO Domain for an organization
   *
   * @name SsoDomainCreate
   * @summary Create Organization SSO Domain
   * @request POST:/api/v1/control-plane/organizations/{organization}/sso-domain
   * @secure
   */
  ssoDomainCreate = (
    organization: string,
    data: CreateOrganizationSsoDomainRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso-domain`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Delete SSO Domain for organization
   *
   * @name SsoDomainDelete
   * @request DELETE:/api/v1/control-plane/organizations/sso-domain/{sso-domain}
   * @secure
   */
  ssoDomainDelete = (ssoDomain: string, params: RequestParams = {}) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/sso-domain/${ssoDomain}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description Get SSO config for organization
   *
   * @name SsoConfigGet
   * @summary List Organization's SSO Domains
   * @request GET:/api/v1/control-plane/organizations/{organization}/sso-config
   * @secure
   */
  ssoConfigGet = (organization: string, params: RequestParams = {}) =>
    this.request<SsoConfig, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso-config`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update SSO config for organization
   *
   * @name SsoConfigUpdate
   * @summary Update organization SSO config
   * @request POST:/api/v1/control-plane/organizations/{organization}/sso-config
   * @secure
   */
  ssoConfigUpdate = (
    organization: string,
    data: SsoConfig,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso-config`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Starts the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateSsoOauthStart
   * @summary Start OAuth flow
   * @request GET:/api/v1/control-plane/users/sso/start
   */
  cloudUserUpdateSsoOauthStart = (
    query: {
      /** The user email */
      email: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/sso/start`,
      method: "GET",
      query: query,
      ...params,
    });
  /**
   * @description Completes the OAuth flow
   *
   * @tags User
   * @name CloudUserUpdateSsoOauthCallback
   * @summary Complete OAuth flow
   * @request GET:/api/v1/control-plane/users/sso/callback
   */
  cloudUserUpdateSsoOauthCallback = (params: RequestParams = {}) =>
    this.request<any, void>({
      path: `/api/v1/control-plane/users/sso/callback`,
      method: "GET",
      ...params,
    });
  /**
   * @description Lists all tenant memberships for the current user
   *
   * @tags User
   * @name TenantMembershipsList
   * @summary List tenant memberships
   * @request GET:/api/v1/control-plane/users/memberships
   * @secure
   */
  tenantMembershipsList = (params: RequestParams = {}) =>
    this.request<UserTenantMembershipsList, APIErrors>({
      path: `/api/v1/control-plane/users/memberships`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all pending tenant invites for the current user
   *
   * @tags Tenant
   * @name UserListTenantInvites
   * @summary List tenant invites
   * @request GET:/api/v1/control-plane/users/tenant-invites
   * @secure
   */
  userListTenantInvites = (params: RequestParams = {}) =>
    this.request<TenantInviteList, APIErrors>({
      path: `/api/v1/control-plane/users/tenant-invites`,
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
   * @request POST:/api/v1/control-plane/users/tenant-invites/accept
   * @secure
   */
  tenantInviteAccept = (
    data: AcceptTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/control-plane/users/tenant-invites/accept`,
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
   * @request POST:/api/v1/control-plane/users/tenant-invites/reject
   * @secure
   */
  tenantInviteReject = (
    data: RejectTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIErrors>({
      path: `/api/v1/control-plane/users/tenant-invites/reject`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      ...params,
    });
  /**
   * @description Generate a signed exchange token for the tenant, embedding the tenant's API URL in the claims
   *
   * @tags Tenant
   * @name ExchangeTokenCreate
   * @summary Generate Tenant Token
   * @request POST:/api/v1/control-plane/tenants/{tenant}/token
   * @secure
   */
  exchangeTokenCreate = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantExchangeToken, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/token`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description List all members of a tenant
   *
   * @tags Tenant
   * @name TenantMemberList
   * @summary List Tenant Members
   * @request GET:/api/v1/control-plane/tenants/{tenant}/members
   * @secure
   */
  tenantMemberList = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantMemberList, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/members`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Update a tenant member's role
   *
   * @tags Tenant
   * @name TenantMemberUpdate
   * @summary Update Tenant Member
   * @request PATCH:/api/v1/control-plane/tenants/{tenant}/members/{tenant-member}
   * @secure
   */
  tenantMemberUpdate = (
    tenant: string,
    tenantMember: string,
    data: UpdateTenantMemberRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantMember, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/members/${tenantMember}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Delete a tenant member
   *
   * @tags Tenant
   * @name TenantMemberDelete
   * @summary Delete Tenant Member
   * @request DELETE:/api/v1/control-plane/tenants/{tenant}/members/{tenant-member}
   * @secure
   */
  tenantMemberDelete = (
    tenant: string,
    tenantMember: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/members/${tenantMember}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all pending invites for a tenant
   *
   * @tags Tenant
   * @name TenantInviteList
   * @summary List Tenant Invites
   * @request GET:/api/v1/control-plane/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteList = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantInviteList, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/invites`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new tenant invite
   *
   * @tags Tenant
   * @name TenantInviteCreate
   * @summary Create Tenant Invite
   * @request POST:/api/v1/control-plane/tenants/{tenant}/invites
   * @secure
   */
  tenantInviteCreate = (
    tenant: string,
    data: CreateTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantInvite, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/invites`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Update a tenant invite's role
   *
   * @tags Tenant
   * @name TenantInviteUpdate
   * @summary Update Tenant Invite
   * @request PATCH:/api/v1/control-plane/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteUpdate = (
    tenant: string,
    tenantInvite: string,
    data: UpdateTenantInviteRequest,
    params: RequestParams = {},
  ) =>
    this.request<TenantInvite, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/invites/${tenantInvite}`,
      method: "PATCH",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Delete a tenant invite
   *
   * @tags Tenant
   * @name TenantInviteDelete
   * @summary Delete Tenant Invite
   * @request DELETE:/api/v1/control-plane/tenants/{tenant}/invites/{tenant-invite}
   * @secure
   */
  tenantInviteDelete = (
    tenant: string,
    tenantInvite: string,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/tenants/${tenant}/invites/${tenantInvite}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
  /**
   * @description List all available subscription plans and their features
   *
   * @tags Billing
   * @name SubscriptionPlansList
   * @summary List subscription plans
   * @request GET:/api/v1/control-plane/billing/plans
   */
  subscriptionPlansList = (params: RequestParams = {}) =>
    this.request<SubscriptionPlanList, APIErrors>({
      path: `/api/v1/control-plane/billing/plans`,
      method: "GET",
      format: "json",
      ...params,
    });
  /**
   * @description Gets the billing state for a tenant
   *
   * @tags Tenant
   * @name TenantBillingStateGet
   * @summary Get the billing state for a tenant
   * @request GET:/api/v1/control-plane/billing/tenants/{tenant}
   * @secure
   */
  tenantBillingStateGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantBillingState, APIErrors | APIError>({
      path: `/api/v1/control-plane/billing/tenants/${tenant}`,
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
   * @summary Update subscription
   * @request PATCH:/api/v1/control-plane/billing/tenants/{tenant}/subscription
   * @secure
   */
  tenantSubscriptionUpdate = (
    tenant: string,
    data: UpdateTenantSubscriptionRequest,
    params: RequestParams = {},
  ) =>
    this.request<UpdateTenantSubscriptionResponse, APIErrors>({
      path: `/api/v1/control-plane/billing/tenants/${tenant}/subscription`,
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
   * @request GET:/api/v1/control-plane/billing/tenants/{tenant}/billing-portal-link
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
      path: `/api/v1/control-plane/billing/tenants/${tenant}/billing-portal-link`,
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
   * @request GET:/api/v1/control-plane/billing/tenants/{tenant}/payment-methods
   * @secure
   */
  tenantPaymentMethodsGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantPaymentMethodList, APIErrors>({
      path: `/api/v1/control-plane/billing/tenants/${tenant}/payment-methods`,
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
   * @request GET:/api/v1/control-plane/billing/tenants/{tenant}/credit-balance
   * @secure
   */
  tenantCreditBalanceGet = (tenant: string, params: RequestParams = {}) =>
    this.request<TenantCreditBalance, APIErrors>({
      path: `/api/v1/control-plane/billing/tenants/${tenant}/credit-balance`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
}
