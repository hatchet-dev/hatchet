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
  CreateManagementTokenRequest,
  CreateManagementTokenResponse,
  CreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest,
  CreateOrganizationRequest,
  CreateOrganizationSsoDomainRequest,
  CreateTenantInviteRequest,
  ManagementTokenList,
  Organization,
  OrganizationForUserList,
  OrganizationInviteList,
  OrganizationTenant,
  RejectOrganizationInviteRequest,
  RejectTenantInviteRequest,
  RemoveOrganizationMembersRequest,
  TenantExchangeToken,
  TenantInvite,
  TenantInviteList,
  TenantMember,
  TenantMemberList,
  UpdateOrganizationRequest,
  UpdateTenantInviteRequest,
  UpdateTenantMemberRequest,
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
   * @name SsoGet
   * @summary List Organization's SSO Configs
   * @request GET:/api/v1/control-plane/organizations/{organization}/sso
   * @secure
   */
  ssoGet = (organization: string, params: RequestParams = {}) =>
    this.request<object, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Create a new organization SSO config
   *
   * @name SsoUpsert
   * @summary Upsert organization SSO config
   * @request POST:/api/v1/control-plane/organizations/{organization}/sso
   * @secure
   */
  ssoUpsert = (
    organization: string,
    data: {
      idpInfoFromCustomer: Record<string, any>;
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
   * @name SsoDomainGet
   * @summary List Organization's SSO Domains
   * @request GET:/api/v1/control-plane/organizations/{organization}/sso_domain
   * @secure
   */
  ssoDomainGet = (organization: string, params: RequestParams = {}) =>
    this.request<
      {
        /** @example "acme.com" */
        sso_domain: string;
      }[],
      APIError
    >({
      path: `/api/v1/control-plane/organizations/${organization}/sso_domain`,
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
   * @request POST:/api/v1/control-plane/organizations/{organization}/sso_domain
   * @secure
   */
  ssoDomainCreate = (
    organization: string,
    data: CreateOrganizationSsoDomainRequest,
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso_domain`,
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
   * @request DELETE:/api/v1/control-plane/organizations/{organization}/sso_domain
   * @secure
   */
  ssoDomainDelete = (
    organization: string,
    data: {
      ssoDomain: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<void, APIError>({
      path: `/api/v1/control-plane/organizations/${organization}/sso_domain`,
      method: "DELETE",
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
}
