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

export enum OrganizationInviteStatus {
  PENDING = "PENDING",
  ACCEPTED = "ACCEPTED",
  REJECTED = "REJECTED",
  EXPIRED = "EXPIRED",
}

export enum ManagementTokenDuration {
  Value30D = "30D",
  Value60D = "60D",
  Value90D = "90D",
}

export enum TenantStatusType {
  ACTIVE = "ACTIVE",
  ARCHIVED = "ARCHIVED",
}

export enum OrganizationMemberRoleType {
  OWNER = "OWNER",
}

export interface APIControlPlaneMetadata {
  /**
   * the inactivity timeout to log out for user sessions in milliseconds
   * @example 3600000
   */
  inactivityLogoutMs?: number;
}

export type { APIErrors } from '@/lib/api/generated/cloud/data-contracts';

export type { APIError } from '@/lib/api/generated/cloud/data-contracts';

export type { PaginationResponse } from '@/lib/api/generated/cloud/data-contracts';

export type { APIResourceMeta } from '@/lib/api/generated/cloud/data-contracts';

export type { User } from '@/lib/api/generated/cloud/data-contracts';

export type { UserLoginRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { UserChangePasswordRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { UserRegisterRequest } from '@/lib/api/generated/cloud/data-contracts';

export interface Organization {
  metadata: APIResourceMeta;
  /** Name of the organization */
  name: string;
  tenants?: OrganizationTenant[];
  members?: OrganizationMember[];
}

export interface OrganizationForUser {
  metadata: APIResourceMeta;
  /** Name of the organization */
  name: string;
  tenants: OrganizationTenant[];
  /** Whether the user is the owner of the organization */
  isOwner: boolean;
}

export interface OrganizationForUserList {
  rows: OrganizationForUser[];
  pagination: PaginationResponse;
}

export interface CreateOrganizationRequest {
  /**
   * Name of the organization
   * @minLength 1
   * @maxLength 256
   */
  name: string;
}

export interface UpdateOrganizationRequest {
  /**
   * Name of the organization
   * @minLength 1
   * @maxLength 256
   */
  name: string;
}

export interface OrganizationMember {
  metadata: APIResourceMeta;
  /** Type/role of the member in the organization */
  role: OrganizationMemberRoleType;
  /**
   * Email of the user
   * @format email
   */
  email: string;
}

export interface OrganizationMemberList {
  rows: OrganizationMember[];
  pagination: PaginationResponse;
}

export interface RemoveOrganizationMembersRequest {
  /**
   * Array of user emails to remove from the organization
   * @minItems 1
   */
  emails: string[];
}

export interface OrganizationTenant {
  /**
   * ID of the tenant
   * @format uuid
   */
  id: string;
  /** Status of the tenant */
  status: TenantStatusType;
  /**
   * The timestamp at which the tenant was archived
   * @format date-time
   */
  archivedAt?: string;
}

export interface OrganizationTenantList {
  rows: OrganizationTenant[];
}

export interface CreateNewTenantForOrganizationRequest {
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
}

export interface CreateManagementTokenRequest {
  /** The name of the management token. */
  name: string;
  /** @default "30D" */
  duration?: ManagementTokenDuration;
}

export interface CreateManagementTokenResponse {
  /** The token of the management token. */
  token: string;
}

export interface ManagementToken {
  /**
   * The ID of the management token.
   * @format uuid
   */
  id: string;
  /** The name of the management token. */
  name: string;
  /**
   * The timestamp at which the management token expires
   * @format date-time
   */
  expiresAt?: string;
}

export interface ManagementTokenList {
  rows: ManagementToken[];
}

export interface OrganizationInvite {
  metadata: APIResourceMeta;
  /**
   * The ID of the organization
   * @format uuid
   */
  organizationId: string;
  /**
   * The email of the inviter
   * @format email
   */
  inviterEmail: string;
  /**
   * The email of the invitee
   * @format email
   */
  inviteeEmail: string;
  /**
   * The timestamp at which the invite expires
   * @format date-time
   */
  expires: string;
  /** The status of the invite */
  status: OrganizationInviteStatus;
  /** The role of the invitee */
  role: OrganizationMemberRoleType;
}

export interface OrganizationInviteList {
  rows: OrganizationInvite[];
}

export interface CreateOrganizationInviteRequest {
  /**
   * The email of the invitee
   * @format email
   */
  inviteeEmail: string;
  /** The role of the invitee */
  role: OrganizationMemberRoleType;
}

export interface AcceptOrganizationInviteRequest {
  /**
   * The ID of the organization invite
   * @format uuid
   */
  id: string;
}

export interface RejectOrganizationInviteRequest {
  /**
   * The ID of the organization invite
   * @format uuid
   */
  id: string;
}

export type { TenantMemberRole } from '@/lib/api/generated/cloud/data-contracts';

export type { UserTenantPublic } from '@/lib/api/generated/cloud/data-contracts';

export type { TenantMember } from '@/lib/api/generated/cloud/data-contracts';

export type { TenantMemberList } from '@/lib/api/generated/cloud/data-contracts';

export type { UpdateTenantMemberRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { TenantInvite } from '@/lib/api/generated/cloud/data-contracts';

export type { TenantInviteList } from '@/lib/api/generated/cloud/data-contracts';

export type { CreateTenantInviteRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { UpdateTenantInviteRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { RejectInviteRequest as RejectTenantInviteRequest } from '@/lib/api/generated/cloud/data-contracts';

export type { UserTenantMembershipsList } from '@/lib/api/generated/cloud/data-contracts';

export interface TenantExchangeToken {
  /** The signed exchange token for the tenant */
  token: string;
  /** The API URL embedded in the token claims */
  apiUrl: string;
  /**
   * Timestamp at which the token expires
   * @format date-time
   */
  expiresAt: string;
}
