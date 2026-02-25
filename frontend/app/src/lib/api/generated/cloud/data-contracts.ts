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

export enum TemplateOptions {
  QUICKSTART_PYTHON = "QUICKSTART_PYTHON",
  QUICKSTART_TYPESCRIPT = "QUICKSTART_TYPESCRIPT",
  QUICKSTART_GO = "QUICKSTART_GO",
}

export enum AutoscalingTargetKind {
  PORTER = "PORTER",
  FLY = "FLY",
}

export enum CouponFrequency {
  Once = "once",
  Recurring = "recurring",
}

export enum ManagedWorkerRegion {
  Ams = "ams",
  Arn = "arn",
  Bom = "bom",
  Cdg = "cdg",
  Dfw = "dfw",
  Ewr = "ewr",
  Fra = "fra",
  Gru = "gru",
  Iad = "iad",
  Jnb = "jnb",
  Lax = "lax",
  Lhr = "lhr",
  Nrt = "nrt",
  Ord = "ord",
  Sin = "sin",
  Sjc = "sjc",
  Syd = "syd",
  Yyz = "yyz",
}

export enum ManagedWorkerEventStatus {
  IN_PROGRESS = "IN_PROGRESS",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELLED = "CANCELLED",
  SCALE_UP = "SCALE_UP",
  SCALE_DOWN = "SCALE_DOWN",
}

export interface APICloudMetadata {
  /**
   * whether the tenant can be billed
   * @example true
   */
  canBill?: boolean;
  /**
   * whether the tenant can link to GitHub
   * @example true
   */
  canLinkGithub?: boolean;
  /**
   * whether metrics are enabled for the tenant
   * @example true
   */
  metricsEnabled?: boolean;
  /**
   * whether the tenant requires billing for managed compute
   * @example true
   */
  requireBillingForManagedCompute?: boolean;
  /**
   * the inactivity timeout to log out for user sessions in milliseconds
   * @example 3600000
   */
  inactivityLogoutMs?: number;
}

export type APIErrors = any;

export type APIError = any;

export type PaginationResponse = any;

export type APIResourceMeta = any;

export interface GithubBranch {
  branch_name: string;
  is_default: boolean;
}

export interface GithubRepo {
  repo_owner: string;
  repo_name: string;
}

export interface GithubAppInstallation {
  type?: "oauth" | "installation";
  metadata: APIResourceMeta;
  installation_settings_url: string;
  account_name: string;
  account_avatar_url: string;
  is_linked_to_tenant: boolean;
}

export interface ListGithubAppInstallationsResponse {
  pagination: PaginationResponse;
  rows: GithubAppInstallation[];
}

export type ListGithubReposResponse = GithubRepo[];

export type ListGithubBranchesResponse = GithubBranch[];

export interface ManagedWorker {
  metadata: APIResourceMeta;
  name: string;
  buildConfig?: ManagedWorkerBuildConfig;
  isIac: boolean;
  directSecrets: ManagedWorkerSecret[];
  globalSecrets: ManagedWorkerSecret[];
  runtimeConfigs?: ManagedWorkerRuntimeConfig[];
  canUpdate?: boolean;
}

export interface ManagedWorkerList {
  rows?: ManagedWorker[];
  pagination?: PaginationResponse;
}

export interface ManagedWorkerBuildConfig {
  metadata: APIResourceMeta;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  githubInstallationId: string;
  githubRepository: GithubRepo;
  githubRepositoryBranch: string;
  steps?: BuildStep[];
}

export interface ManagedWorkerSecret {
  key: string;
  id: string;
  hint: string;
}

export interface CreateManagedWorkerSecretRequest {
  /** array of secret keys and values to add to the worker */
  add?: {
    key: string;
    value: string;
  }[];
  /** array of global secret ids to add to the worker */
  addGlobal?: string[];
}

export interface UpdateManagedWorkerSecretRequest {
  /** array of secret keys and values to add to the worker */
  add?: {
    key: string;
    value: string;
  }[];
  /** array of global secret ids to add to the worker */
  addGlobal?: string[];
  /** array of secret ids to delete from the worker */
  delete?: string[];
  /** array of existing secret ids and values to update in the worker */
  update?: {
    /**
     * @format uuid
     * @minLength 36
     * @maxLength 36
     */
    id: string;
    /** @minLength 1 */
    key: string;
    value: string;
  }[];
}

export interface BuildStep {
  metadata: APIResourceMeta;
  /** The relative path to the build directory */
  buildDir: string;
  /** The relative path from the build dir to the Dockerfile */
  dockerfilePath: string;
}

export interface ManagedWorkerRuntimeConfig {
  metadata: APIResourceMeta;
  numReplicas: number;
  autoscaling?: AutoscalingConfig;
  /** The kind of CPU to use for the worker */
  cpuKind: string;
  /** The number of CPUs to use for the worker */
  cpus: number;
  /** The amount of memory in MB to use for the worker */
  memoryMb: number;
  /** The region that the worker is deployed to */
  region: ManagedWorkerRegion;
  /** A list of actions this runtime config corresponds to */
  actions?: string[];
}

export interface ManagedWorkerEvent {
  id: number;
  /** @format date-time */
  timeFirstSeen: string;
  /** @format date-time */
  timeLastSeen: string;
  managedWorkerId: string;
  status: ManagedWorkerEventStatus;
  message: string;
  data: object;
}

export interface ManagedWorkerEventList {
  pagination?: PaginationResponse;
  rows?: ManagedWorkerEvent[];
}

export interface CreateManagedWorkerRequest {
  name: string;
  buildConfig: CreateManagedWorkerBuildConfigRequest;
  secrets?: CreateManagedWorkerSecretRequest;
  isIac: boolean;
  runtimeConfig?: CreateManagedWorkerRuntimeConfigRequest;
}

export interface UpdateManagedWorkerRequest {
  name?: string;
  buildConfig?: CreateManagedWorkerBuildConfigRequest;
  secrets?: UpdateManagedWorkerSecretRequest;
  isIac?: boolean;
  runtimeConfig?: CreateManagedWorkerRuntimeConfigRequest;
}

export interface InfraAsCodeRequest {
  runtimeConfigs: CreateManagedWorkerRuntimeConfigRequest[];
}

export interface RuntimeConfigActionsResponse {
  actions: string[];
}

export interface CreateManagedWorkerBuildConfigRequest {
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   */
  githubInstallationId: string;
  githubRepositoryOwner: string;
  githubRepositoryName: string;
  githubRepositoryBranch: string;
  steps: CreateBuildStepRequest[];
}

export interface CreateBuildStepRequest {
  /** The relative path to the build directory */
  buildDir: string;
  /** The relative path from the build dir to the Dockerfile */
  dockerfilePath: string;
}

export interface CreateManagedWorkerRuntimeConfigRequest {
  /**
   * @min 0
   * @max 1000
   */
  numReplicas?: number;
  autoscaling?: CreateOrUpdateAutoscalingRequest;
  /** The region to deploy the worker to */
  regions?: ManagedWorkerRegion[];
  /** The kind of CPU to use for the worker */
  cpuKind: string;
  /**
   * The number of CPUs to use for the worker
   * @min 1
   * @max 64
   */
  cpus: number;
  /**
   * The amount of memory in MB to use for the worker
   * @min 1024
   * @max 65536
   */
  memoryMb: number;
  /** The kind of GPU to use for the worker */
  gpuKind?: "a10" | "l40s" | "a100-40gb" | "a100-80gb";
  /**
   * The number of GPUs to use for the worker
   * @min 1
   * @max 8
   */
  gpus?: number;
  actions?: string[];
  /**
   * @min 1
   * @max 1000
   */
  slots?: number;
}

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
  period?: string;
}

export interface UpdateTenantSubscriptionRequest {
  /** The code of the plan. */
  plan: string;
  /** The period of the plan. */
  period?: string;
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
  plan: string;
  /** The period associated with the tenant subscription. */
  period?: string;
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

export type VectorPushRequest = EventObject[];

export interface EventObject {
  event?: {
    provider?: string;
  };
  fly?: {
    app?: {
      instance?: string;
      name?: string;
    };
    region?: string;
  };
  host?: string;
  log?: {
    level?: string;
  };
  message?: string;
  /** @format date-time */
  timestamp?: string;
}

export interface LogLine {
  /** @format date-time */
  timestamp: string;
  instance: string;
  line: string;
  metadata?: object;
  retryCount?: number;
  level?: string;
}

export interface LogLineList {
  rows?: LogLine[];
  pagination?: PaginationResponse;
}

export type Matrix = SampleStream[];

export interface SampleStream {
  metric?: Metric;
  values?: SamplePair[];
  histograms?: SampleHistogramPair[];
}

export type SamplePair = any[];

/** @format float */
export type SampleValue = number;

export interface SampleHistogramPair {
  timestamp?: Time;
  histogram?: SampleHistogram;
}

export interface SampleHistogram {
  count?: FloatString;
  sum?: FloatString;
  buckets?: HistogramBuckets;
}

/** @format float */
export type FloatString = number;

export type HistogramBuckets = HistogramBucket[];

export interface HistogramBucket {
  /** @format int32 */
  boundaries?: number;
  lower?: FloatString;
  upper?: FloatString;
  count?: FloatString;
}

export type Metric = Record<string, LabelValue>;

export type LabelSet = Record<string, LabelValue>;

export type LabelName = string;

export type LabelValue = string;

export type Time = number;

export interface Build {
  metadata?: APIResourceMeta;
  status: string;
  statusDetail?: string;
  /** @format date-time */
  createTime: string;
  /** @format date-time */
  startTime?: string;
  /** @format date-time */
  finishTime?: string;
  /** @format uuid */
  buildConfigId: string;
}

export interface Instance {
  instanceId: string;
  name: string;
  region: string;
  state: string;
  cpuKind: string;
  cpus: number;
  memoryMb: number;
  diskGb: number;
  commitSha: string;
}

export interface InstanceList {
  pagination?: PaginationResponse;
  rows?: Instance[];
}

/**
 * a map of feature flags for the tenant
 * @example {"flag1":"value1","flag2":"value2"}
 */
export type FeatureFlags = Record<string, string>;

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

export interface AutoscalingConfig {
  waitDuration: string;
  rollingWindowDuration: string;
  utilizationScaleUpThreshold: number;
  utilizationScaleDownThreshold: number;
  increment: number;
  targetKind: AutoscalingTargetKind;
  minAwakeReplicas: number;
  maxReplicas: number;
  scaleToZero: boolean;
}

export interface CreateOrUpdateAutoscalingRequest {
  waitDuration: string;
  rollingWindowDuration: string;
  utilizationScaleUpThreshold: number;
  utilizationScaleDownThreshold: number;
  increment: number;
  targetKind?: AutoscalingTargetKind;
  minAwakeReplicas: number;
  maxReplicas: number;
  scaleToZero: boolean;
  porter?: CreatePorterAutoscalingRequest;
  fly?: CreateFlyAutoscalingRequest;
}

export interface CreatePorterAutoscalingRequest {
  token: string;
  targetUrl: "CLOUD" | "DASHBOARD";
  targetProject: string;
  targetCluster: string;
  targetAppName: string;
}

export interface CreateFlyAutoscalingRequest {
  autoscalingKey: string;
  currentReplicas: number;
}

export interface CreateManagedWorkerFromTemplateRequest {
  name: TemplateOptions;
}

export interface MonthlyComputeCost {
  cost: number;
  hasCreditsRemaining: boolean;
  creditsRemaining?: number;
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
