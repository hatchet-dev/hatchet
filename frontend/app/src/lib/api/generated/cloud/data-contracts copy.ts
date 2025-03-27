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

export interface APICloudMetadata {
  /**
   * whether the tenant can be billed
   * @example true
   */
  canBill?: boolean;
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
  buildConfig: ManagedWorkerBuildConfig;
  runtimeConfig: ManagedWorkerRuntimeConfig;
}

export interface ManagedWorkerBuildConfig {
  metadata: APIResourceMeta;
  githubRepository: GithubRepo;
  githubRepositoryBranch: string;
  steps: BuildStep[];
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
  /** A map of environment variables to set for the worker */
  envVars: Record<string, string>;
  /** The kind of CPU to use for the worker */
  cpuKind: string;
  /** The number of CPUs to use for the worker */
  cpus: number;
  /** The amount of memory in MB to use for the worker */
  memoryMb: number;
}

export interface CreateManagedWorkerRequest {
  name: string;
  buildConfig: CreateManagedWorkerBuildConfigRequest;
  runtimeConfig: CreateManagedWorkerRuntimeConfigRequest;
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
   * @min 1
   * @max 16
   */
  numReplicas: number;
  /** A map of environment variables to set for the worker */
  envVars: Record<string, string>;
  /** The kind of CPU to use for the worker */
  cpuKind: string;
  /** The number of CPUs to use for the worker */
  cpus: number;
  /** The amount of memory in MB to use for the worker */
  memoryMb: number;
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

export enum TenantSubscriptionStatus {
  Active = "active",
  Pending = "pending",
  Terminated = "terminated",
  Canceled = "canceled",
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

export enum CouponFrequency {
  Once = "once",
  Recurring = "recurring",
}
