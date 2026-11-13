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

export enum SandboxProviderEventKind {
  HEALTHCHECK_FAILED = "HEALTHCHECK_FAILED",
  UNHEALTHY = "UNHEALTHY",
  HEALTHY = "HEALTHY",
  SANDBOX_CREATED = "SANDBOX_CREATED",
  SANDBOX_CREATE_FAILED = "SANDBOX_CREATE_FAILED",
  TURN_FAILED = "TURN_FAILED",
  SNAPSHOT_FAILED = "SNAPSHOT_FAILED",
  SANDBOX_DESTROYED = "SANDBOX_DESTROYED",
}

/** The sandbox vendor a provider connects to. */
export enum SandboxProviderKind {
  DAYTONA = "DAYTONA",
  E2B = "E2B",
  ARCHIL = "ARCHIL",
  FREESTYLE = "FREESTYLE",
}

export enum UserOfferType {
  YCAlumni = "YC Alumni",
  YCCurrentBatch = "YC Current Batch",
  Startup = "Startup",
  Custom = "Custom",
}

export enum UserOfferStage {
  Requested = "Requested",
  Approved = "Approved",
  Redeemed = "Redeemed",
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
  MEMBER = "MEMBER",
}

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

export interface APICloudMetadata {
  /**
   * whether the tenant can be billed
   * @example true
   */
  canBill?: boolean;
  /**
   * whether metrics are enabled for the tenant
   * @example true
   */
  metricsEnabled?: boolean;
  /**
   * the inactivity timeout to log out for user sessions in milliseconds
   * @example 3600000
   */
  inactivityLogoutMs?: number;
  /**
   * when set, the dashboard may prompt users to continue on this frontend host (same path)
   * @example "https://cloud.hatchet.run"
   */
  redirectFrontendHost?: string;
}

export type APIErrors = any;

export type APIError = any;

export type PaginationResponse = any;

export type APIResourceMeta = any;

export interface TenantBillingState {
  /** The subscription associated with this policy. */
  currentSubscription: TenantSubscription;
  /** The upcoming subscription associated with this policy. */
  upcomingSubscription?: TenantSubscription;
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
  /**
   * The expiration date of the card.
   * @format date-time
   */
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

export type UpdateTenantSubscriptionResponse =
  | CheckoutURLResponse
  | {
      currentSubscription: TenantSubscription;
      upcomingSubscription?: TenantSubscription;
    };

export interface CheckoutURLResponse {
  /** The URL to the checkout page. */
  checkoutUrl: string;
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

export interface WorkflowRunEventsMetric {
  /** @format date-time */
  time: string;
  PENDING: number;
  RUNNING: number;
  SUCCEEDED: number;
  FAILED: number;
  QUEUED: number;
}

export interface WorkflowRunEventsMetricsCounts {
  results?: WorkflowRunEventsMetric[];
}

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

export interface InviteOrganizationMembersRequest {
  /**
   * Array of user emails to invite to the organization
   * @minItems 1
   */
  emails: string[];
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

export interface UpdateOrganizationTenantRequest {
  /** The name of the tenant. */
  name: string;
}

export type APIToken = any;

export type APITokenList = any;

export type CreateTenantAPITokenRequest = any;

export type CreateTenantAPITokenResponse = any;

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
  /** The name of the organization */
  organizationName?: string;
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

export interface RedeemOfferRequest {
  /**
   * The organization to apply the offer credit to.
   * @format uuid
   */
  organizationId: string;
  /** The Attio record ID of the offer to redeem. */
  offerRecordId: string;
  /** Shipping address for welcome kit delivery. */
  address?: string;
}

export interface RedeemOfferResponse {
  /** The amount of credit applied in cents. */
  appliedCents: number;
  /** The Attio record ID of the redeemed offer. */
  offerRecordId: string;
}

export interface UserOffer {
  /** The Attio record ID of the offer. */
  recordId: string;
  /** The current stage of the offer. */
  stage?: UserOfferStage;
  /** The type of the offer. */
  type?: UserOfferType;
  /** The credit amount in cents. */
  creditAmountCents?: number;
  /**
   * The expiration date of the offer.
   * @format date-time
   */
  expiresAt?: string;
  /** Number of months until the offer credit expires after redemption. */
  expiresInMonths?: number;
  /** Whether the offer includes a welcome kit. */
  includesWelcomeKit?: boolean;
}

export type AutumnWebhookEvent = AutumnCustomerProductsUpdatedEvent;

export interface AutumnCustomerProductsUpdatedEvent {
  data: AutumnCustomerProductsUpdatedEventData;
  type: string;
}

export interface AutumnCustomerProductsUpdatedEventData {
  customer: AutumnCustomer;
  entity: {
    /** @format int64 */
    created_at?: number;
    customer_id: string;
    env?: string;
    features?: AutumnFeaturesMap;
    id: string;
    name?: string;
    products: AutumnCustomerProduct[];
  };
  scenario?: string;
  updated_product?: {
    archived?: boolean;
    base_variant_id?: string;
    /** @format int64 */
    created_at?: number;
    env?: string;
    free_trial?: object;
    group?: string;
    id: string;
    is_add_on?: boolean;
    is_default?: boolean;
    items?: AutumnProductItem[];
    name?: string;
    properties?: {
      has_trial?: boolean;
      interval_group?: string;
      is_free?: boolean;
      is_one_off?: boolean;
      updateable?: boolean;
    };
    version?: number;
  };
}

export interface AutumnCustomer {
  autumn_id?: string;
  /** @format int64 */
  created_at?: number;
  email?: string;
  env?: string;
  features?: AutumnFeaturesMap;
  fingerprint?: string;
  id: string;
  metadata: Record<string, any>;
  name: string;
  products?: AutumnCustomerProduct[];
  send_email_receipts?: boolean;
  stripe_id?: string;
}

export interface AutumnCustomerProduct {
  /** @format int64 */
  canceled_at?: number;
  /** @format int64 */
  current_period_end?: number;
  /** @format int64 */
  current_period_start?: number;
  group?: string;
  id: string;
  is_add_on?: boolean;
  is_default?: boolean;
  items?: AutumnProductItem[];
  name?: string;
  quantity?: number;
  /** @format int64 */
  started_at?: number;
  status?: string;
  version?: number;
}

export type AutumnFeaturesMap = Record<string, AutumnFeature>;

export interface AutumnFeature {
  balance?: number;
  breakdown?: AutumnFeatureBreakdown[];
  credit_schema?: AutumnFeatureCreditSchemaItem[];
  id: string;
  included_usage?: number;
  interval?: string;
  interval_count?: number;
  name: string;
  /** @format int64 */
  next_reset_at?: number;
  overage_allowed?: boolean;
  type: string;
  unlimited?: boolean;
  usage?: number;
}

export interface AutumnFeatureBreakdown {
  balance?: number;
  /** @format int64 */
  expires_at?: number;
  included_usage?: number;
  interval?: string;
  interval_count?: number;
  /** @format int64 */
  next_reset_at?: number;
  overage_allowed?: boolean;
  usage?: number;
}

export interface AutumnFeatureCreditSchemaItem {
  credit_amount: number;
  feature_id: string;
}

export interface AutumnProductItem {
  billing_units?: number;
  display?: AutumnProductItemDisplay;
  entity_feature_id?: string;
  feature?: {
    archived?: boolean;
    credit_schema?: {
      credit_cost: number;
      metered_feature_id: string;
    }[];
    display?: {
      plural?: string;
      singular?: string;
    };
    id: string;
    name: string;
    type: string;
  };
  feature_id?: string;
  feature_type?: string;
  included_usage?: number;
  interval?: string;
  interval_count?: number;
  price?: number;
  reset_usage_when_enabled?: boolean;
  type: string;
  usage_model?: string;
}

export interface AutumnProductItemDisplay {
  primary_text?: string;
  secondary_text?: string;
}

/** Request body for adding existing org members to a specific tenant, bypassing tag matching. */
export interface AddOrganizationMembersToTenantRequest {
  /** IDs of org members to add to the tenant. */
  memberIds: string[];
}

/** The health of the provider as last observed by the sandbox operator: whether the vendor accepted the provider's credentials. Written on transitions only. */
export interface SandboxProviderStatus {
  /** Whether the last credential check succeeded. Absent until the operator has checked the provider. */
  healthy?: boolean;
  /** The error from the last failed check, if the provider is unhealthy. */
  error?: string;
  /**
   * When the healthy flag last changed.
   * @format date-time
   */
  changedAt?: string;
}

export interface SandboxProvider {
  metadata: APIResourceMeta;
  /** @format uuid */
  tenantId: string;
  /** The name of the provider. Unique within the tenant; the SDK selects it with metadata.provider. */
  name: string;
  /** The sandbox vendor a provider connects to. */
  kind: SandboxProviderKind;
  /** Provider settings as JSON. */
  config: Record<string, any>;
  /** The template or image a sandbox is created from when the SDK names none. */
  defaultTemplate?: string;
  /**
   * The longest idle time a sandbox may ask for before the vendor removes it, in seconds.
   * @format int32
   */
  maxIdleTtlSeconds: number;
  labels: Record<string, any>;
  /** Whether the operator serves sandboxes on this provider. */
  enabled: boolean;
  /** The health of the provider as last observed by the sandbox operator: whether the vendor accepted the provider's credentials. Written on transitions only. */
  status: SandboxProviderStatus;
}

export interface SandboxProviderList {
  pagination?: PaginationResponse;
  rows?: SandboxProvider[];
}

export interface CreateSandboxProviderRequest {
  /**
   * The name of the provider. Unique within the tenant.
   * @minLength 1
   * @maxLength 255
   */
  name: string;
  /** The sandbox vendor a provider connects to. */
  kind: SandboxProviderKind;
  /**
   * The vendor API key. Write-only; it is stored encrypted and never returned.
   * @minLength 1
   */
  apiKey: string;
  /** Provider settings as JSON. Unknown keys are rejected. */
  config?: Record<string, any>;
  defaultTemplate?: string;
  /**
   * Defaults to 900.
   * @format int32
   */
  maxIdleTtlSeconds?: number;
  labels?: Record<string, any>;
  /** Defaults to true. */
  enabled?: boolean;
}

/** Fields to update on a sandbox provider. Omitted fields are left unchanged. */
export interface UpdateSandboxProviderRequest {
  /**
   * @minLength 1
   * @maxLength 255
   */
  name?: string;
  /**
   * A new vendor API key. Provide a value to rotate the key; it is never returned.
   * @minLength 1
   */
  apiKey?: string;
  /** Replaces the existing config. */
  config?: Record<string, any>;
  defaultTemplate?: string;
  /** @format int32 */
  maxIdleTtlSeconds?: number;
  /** Replaces the existing labels. */
  labels?: Record<string, any>;
  enabled?: boolean;
}

export interface SandboxProviderEvent {
  /** @format int64 */
  id: number;
  /** @format date-time */
  createdAt: string;
  kind: SandboxProviderEventKind;
  sandboxId?: string;
  message: string;
  details: Record<string, any>;
}

export interface SandboxProviderEventList {
  rows: SandboxProviderEvent[];
}
