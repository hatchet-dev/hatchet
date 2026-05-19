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

export enum CouponFrequency {
  Once = "once",
  Recurring = "recurring",
}

export enum SubscriptionPeriod {
  Monthly = "monthly",
  Yearly = "yearly",
}

export enum SubscriptionPlanCode {
  Free = "free",
  Starter = "starter",
  Growth = "growth",
  Developer = "developer",
  Team = "team",
  Scale = "scale",
  Dedicated = "dedicated",
}

/** SHARED when the shard is in the general pool; DEDICATED when it is pinned to specific organizations. */
export enum OrganizationAvailableShardClass {
  SHARED = "SHARED",
  DEDICATED = "DEDICATED",
}

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
  auth?: APIMetaAuth;
  /**
   * the Pylon app ID for usepylon.com chat support
   * @example "12345678-1234-1234-1234-123456789012"
   */
  pylonAppId?: string;
  posthog?: APIMetaPosthog;
  /**
   * whether or not users can sign up for this instance
   * @example true
   */
  allowSignup?: boolean;
  /**
   * whether or not users can invite other users to this instance
   * @example true
   */
  allowInvites?: boolean;
  /**
   * whether or not users can create new tenants
   * @example true
   */
  allowCreateTenant?: boolean;
  /**
   * whether or not users can change their password
   * @example true
   */
  allowChangePassword?: boolean;
  /**
   * whether or not observability (trace collection) is enabled on this instance
   * @example false
   */
  observabilityEnabled?: boolean;
}

import type { APIMetaAuth } from '@/lib/api/generated/data-contracts';

import type { APIMetaPosthog } from '@/lib/api/generated/data-contracts';

import type { APIErrors } from '@/lib/api/generated/data-contracts';

import type { APIError } from '@/lib/api/generated/data-contracts';

import type { PaginationResponse } from '@/lib/api/generated/data-contracts';

import type { APIResourceMeta } from '@/lib/api/generated/data-contracts';

export type ListAPIMetaIntegration = APIMetaIntegration[];

import type { APIMetaIntegration } from '@/lib/api/generated/data-contracts';

import type { User } from '@/lib/api/generated/data-contracts';

import type { UserLoginRequest } from '@/lib/api/generated/data-contracts';

import type { UserChangePasswordRequest } from '@/lib/api/generated/data-contracts';

import type { UserRegisterRequest } from '@/lib/api/generated/data-contracts';

export interface Organization {
  metadata: APIResourceMeta;
  /** Name of the organization */
  name: string;
  tenants?: OrganizationTenant[];
  members?: OrganizationMember[];
  /**
   * Time of inactivity to force log out a user (ms)
   * @format int64
   */
  inactivity_timeout?: number;
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
  name?: string;
  /** Inactivity timeout */
  inactivity_timeout?: string;
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
  /** Name of the tenant */
  name?: string;
  /** Slug of the tenant */
  slug?: string;
  /** Status of the tenant */
  status: TenantStatusType;
  /**
   * The timestamp at which the tenant was archived
   * @format date-time
   */
  archivedAt?: string;
  /** Control-plane shard region for the tenant (e.g. aws:us-west-2). */
  region?: string;
}

export interface OrganizationTenantList {
  rows: OrganizationTenant[];
}

export interface CreateNewTenantForOrganizationRequest {
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
  /**
   * Optional shard region (e.g. aws:us-east-1). When omitted, the server picks one.
   * @example "aws:us-east-1"
   */
  region?: string;
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

export interface CreateOrganizationSsoDomainRequest {
  /** @format uri */
  ssoDomain: string;
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

import type { TenantMemberRole } from '@/lib/api/generated/data-contracts';

import type { UserTenantPublic } from '@/lib/api/generated/data-contracts';

import type { TenantMember } from '@/lib/api/generated/data-contracts';

import type { TenantMemberList } from '@/lib/api/generated/data-contracts';

import type { UpdateTenantMemberRequest } from '@/lib/api/generated/data-contracts';

import type { TenantInvite } from '@/lib/api/generated/data-contracts';

import type { TenantInviteList } from '@/lib/api/generated/data-contracts';

import type { CreateTenantInviteRequest } from '@/lib/api/generated/data-contracts';

import type { UpdateTenantInviteRequest } from '@/lib/api/generated/data-contracts';

import type { AcceptInviteRequest as AcceptTenantInviteRequest } from '@/lib/api/generated/data-contracts';

import type { RejectInviteRequest as RejectTenantInviteRequest } from '@/lib/api/generated/data-contracts';

import type { UserTenantMembershipsList } from '@/lib/api/generated/data-contracts';

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

export interface APIToken {
  metadata: APIResourceMeta;
  /** The name of the API token */
  name: string;
  /**
   * The timestamp at which the token expires
   * @format date-time
   */
  expiresAt: string;
}

export interface APITokenList {
  rows: APIToken[];
  pagination?: PaginationResponse;
}

export interface CreateTenantAPITokenRequest {
  /** The name of the API token */
  name: string;
  /** The duration for which the token should be valid (e.g., "30d", "90d") */
  expiresIn?: string;
}

export interface CreateTenantAPITokenResponse {
  /** The generated API token */
  token: string;
}

export interface OrganizationAvailableShard {
  /** Cloud provider for this deployment target (e.g. aws). */
  provider: string;
  /** Region within the provider (e.g. us-east-1). */
  region: string;
  /** SHARED when the shard is in the general pool; DEDICATED when it is pinned to specific organizations. */
  shardClass: OrganizationAvailableShardClass;
}

export interface OrganizationAvailableShardList {
  rows: OrganizationAvailableShard[];
}

export interface SsoDomain {
  /**
   * @format uri
   * @example "acme.com"
   */
  ssoDomain: string;
  /** @example false */
  verified: boolean;
  /** @format uuid */
  verificationToken: string;
}

export type SsoDomainArray = SsoDomain[];

export interface SsoConfig {
  /** @example false */
  forceSSO: boolean;
}

export interface TenantBillingState {
  /** The subscription associated with this policy. */
  currentSubscription: TenantBillingStateSubscription;
  /** The upcoming subscription associated with this policy. */
  upcomingSubscription?: TenantBillingStateSubscription;
  /** A list of plans available for the tenant. */
  plans: SubscriptionPlan[];
  /** A list of coupons applied to the tenant. */
  coupons?: Coupon[];
}

export interface TenantPaymentMethod {
  /** The brand of the payment method. */
  brand: string;
  /** The last 4 digits of the card. */
  last4?: string;
  /** The expiration date of the card. */
  expiration?: string;
  /** The description of the payment method. */
  description?: string;
}

export type TenantPaymentMethodList = TenantPaymentMethod[];

export interface TenantCreditBalance {
  /** The Stripe customer balance in cents. Negative means customer credit. */
  balanceCents: number;
  /** ISO currency code for the Stripe customer balance. */
  currency: string;
  /** Human-readable description for the active credit balance, if available. */
  description?: string;
  /**
   * The timestamp at which the current credit balance is scheduled to expire.
   * @format date-time
   */
  expiresAt?: string;
}

export interface TenantSubscription {
  /** The plan code associated with the tenant subscription. */
  plan: SubscriptionPlanCode;
  /** The period associated with the tenant subscription. */
  period?: SubscriptionPeriod;
  /**
   * The start date of the tenant subscription.
   * @format date-time
   */
  startedAt: string;
  /**
   * The end date of the tenant subscription.
   * @format date-time
   */
  endsAt?: string;
}

export interface TenantBillingStateSubscription {
  /** The plan code associated with the tenant subscription. */
  plan: SubscriptionPlanCode;
  /** The period associated with the tenant subscription. */
  period?: SubscriptionPeriod;
  /**
   * The start date of the tenant subscription.
   * @format date-time
   */
  startedAt: string;
  /**
   * The end date of the tenant subscription.
   * @format date-time
   */
  endsAt?: string;
}

export interface UpdateTenantSubscriptionState {
  /** The plan code associated with the tenant subscription. */
  plan: SubscriptionPlanCode;
  /** The period associated with the tenant subscription. */
  period?: SubscriptionPeriod;
  /**
   * The start date of the tenant subscription.
   * @format date-time
   */
  startedAt: string;
  /**
   * The end date of the tenant subscription.
   * @format date-time
   */
  endsAt?: string;
}

export interface SubscriptionPlanFeatureDisplay {
  /** Main display text for this feature (e.g. "100,000 task runs"). */
  primaryText: string;
  /** Secondary display text (e.g. "then $10 per 1,000,000 task runs"). */
  secondaryText?: string;
}

export interface SubscriptionPlanFeatureOverage {
  /**
   * Price per billing units of overage usage.
   * @format double
   */
  price: number;
  /**
   * Number of units per price increment.
   * @format int64
   */
  billingUnits: number;
  /** How overage is charged (e.g. "pay_per_use", "prepaid"). */
  usageModel: string;
}

export interface SubscriptionPlanFeature {
  /** The identifier of the feature. */
  featureId: string;
  /** Human-readable name of the feature. */
  name: string;
  /** The type of the feature (e.g. "boolean", "single_use", "continuous_use"). */
  featureType: string;
  /** Whether this feature is part of this plan. False for features added for cross-plan comparison. */
  included: boolean;
  /**
   * The included usage for this feature in the plan.
   * @format int64
   */
  includedUsage: number;
  /** Whether this feature has unlimited usage. */
  unlimited: boolean;
  /** Overage pricing details, if applicable. */
  overage?: SubscriptionPlanFeatureOverage;
  /** Pre-formatted display text for this feature. */
  display?: SubscriptionPlanFeatureDisplay;
}

export interface SubscriptionPlanFeatureGroup {
  /** The name of the feature group (e.g. "Usage", "Infrastructure"). */
  name: string;
  /** The features in this group. */
  features: SubscriptionPlanFeature[];
}

export interface SubscriptionPlan {
  /** The code of the plan. */
  planCode: string;
  /** The name of the plan. */
  name: string;
  /** The description of the plan. */
  description: string;
  /** The price of the plan. */
  amountCents: number;
  /** The period of the plan. */
  period?: SubscriptionPeriod;
  /** Whether this is a legacy plan and is no longer offered to new customers. */
  legacy?: boolean;
  /** The features included in this plan, organized by group. */
  featureGroups?: SubscriptionPlanFeatureGroup[];
}

export interface SubscriptionPlanFreeLimit {
  /** The feature identifier. */
  featureId: string;
  /** Human-readable name of the limit. */
  name: string;
  /**
   * The daily limit value.
   * @format int64
   */
  limit: number;
}

export interface SubscriptionPlanList {
  plans: SubscriptionPlan[];
  /** Abbreviated daily limits for the free plan. */
  freeLimits: SubscriptionPlanFreeLimit[];
}

export interface UpdateTenantSubscriptionRequest {
  /** The code of the plan. */
  plan: SubscriptionPlanCode;
  /** The period of the plan. */
  period?: SubscriptionPeriod;
}

export interface UpdateTenantSubscriptionResponse {
  /** The URL to the checkout page. */
  checkoutUrl?: string;
  currentSubscription?: UpdateTenantSubscriptionState;
  upcomingSubscription?: UpdateTenantSubscriptionState;
}

export interface CheckoutURLResponse {
  /** The URL to the checkout page. */
  checkoutUrl: string;
}

export interface Coupon {
  /** The name of the coupon. */
  name: string;
  /** The amount off of the coupon. */
  amount_cents?: number;
  /** The amount remaining on the coupon. */
  amount_cents_remaining?: number;
  /** The currency of the coupon. */
  amount_currency?: string;
  /** The frequency of the coupon. */
  frequency: CouponFrequency;
  /** The frequency duration of the coupon. */
  frequency_duration?: number;
  /** The frequency duration remaining of the coupon. */
  frequency_duration_remaining?: number;
  /** The percentage off of the coupon. */
  percent?: number;
}
