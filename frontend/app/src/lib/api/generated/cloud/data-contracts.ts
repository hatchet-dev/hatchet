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

export enum TenantSubscriptionStatus {
  Active = "active",
  Pending = "pending",
  Terminated = "terminated",
  Canceled = "canceled",
}

export enum ManagedWorkerRegion {
  Ams = "ams",
  Arn = "arn",
  Atl = "atl",
  Bog = "bog",
  Bos = "bos",
  Cdg = "cdg",
  Den = "den",
  Dfw = "dfw",
  Ewr = "ewr",
  Eze = "eze",
  Gdl = "gdl",
  Gig = "gig",
  Gru = "gru",
  Hkg = "hkg",
  Iad = "iad",
  Jnb = "jnb",
  Lax = "lax",
  Lhr = "lhr",
  Mad = "mad",
  Mia = "mia",
  Nrt = "nrt",
  Ord = "ord",
  Otp = "otp",
  Phx = "phx",
  Qro = "qro",
  Scl = "scl",
  Sea = "sea",
  Sin = "sin",
  Sjc = "sjc",
  Syd = "syd",
  Waw = "waw",
  Yul = "yul",
  Yyz = "yyz",
}

export enum ManagedWorkerEventStatus {
  IN_PROGRESS = "IN_PROGRESS",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELLED = "CANCELLED",
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
}

export interface APIErrors {
  errors: APIError[];
}

export interface APIError {
  /**
   * a custom Hatchet error code
   * @format uint64
   * @example 1400
   */
  code?: number;
  /**
   * the field that this error is associated with, if applicable
   * @example "name"
   */
  field?: string;
  /**
   * a description for this error
   * @example "A descriptive error message"
   */
  description: string;
  /**
   * a link to the documentation for this error, if it exists
   * @example "github.com/hatchet-dev/hatchet"
   */
  docs_link?: string;
}

/** @example {"next_page":3,"num_pages":10,"current_page":2} */
export interface PaginationResponse {
  /**
   * the current page
   * @format int64
   * @example 2
   */
  current_page?: number;
  /**
   * the next page
   * @format int64
   * @example 3
   */
  next_page?: number;
  /**
   * the total number of pages for listing
   * @format int64
   * @example 10
   */
  num_pages?: number;
}

export interface APIResourceMeta {
  /**
   * the id of this resource, in UUID format
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  id: string;
  /**
   * the time that this resource was created
   * @format date-time
   * @example "2022-12-13T20:06:48.888Z"
   */
  createdAt: string;
  /**
   * the time that this resource was last updated
   * @format date-time
   * @example "2022-12-13T20:06:48.888Z"
   */
  updatedAt: string;
}

export interface GithubBranch {
  branch_name: string;
  is_default: boolean;
}

export interface GithubRepo {
  repo_owner: string;
  repo_name: string;
}

export interface GithubAppInstallation {
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
  paymentMethods?: TenantPaymentMethod[];
  /** The subscription associated with this policy. */
  subscription: TenantSubscription;
  /** A list of plans available for the tenant. */
  plans?: SubscriptionPlan[];
  /** A list of coupons applied to the tenant. */
  coupons?: Coupon[];
}

export interface SubscriptionPlan {
  /** The code of the plan. */
  plan_code: string;
  /** The name of the plan. */
  name: string;
  /** The description of the plan. */
  description: string;
  /** The price of the plan. */
  amount_cents: number;
  /** The period of the plan. */
  period?: string;
}

export interface UpdateTenantSubscription {
  /** The code of the plan. */
  plan?: string;
  /** The period of the plan. */
  period?: string;
}

export interface TenantSubscription {
  /** The plan code associated with the tenant subscription. */
  plan?: string;
  /** The period associated with the tenant subscription. */
  period?: string;
  /** The status of the tenant subscription. */
  status?: TenantSubscriptionStatus;
  /** A note associated with the tenant subscription. */
  note?: string;
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
