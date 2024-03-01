/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface APIMeta {
  auth?: APIMetaAuth;
}

export interface APIMetaAuth {
  /**
   * the supported types of authentication
   * @example ["basic","google"]
   */
  schemes?: string[];
}

export type ListAPIMetaIntegration = APIMetaIntegration[];

export interface APIMetaIntegration {
  /**
   * the name of the integration
   * @example "github"
   */
  name: string;
  /** whether this integration is enabled on the instance */
  enabled: boolean;
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

export interface User {
  metadata: APIResourceMeta;
  /** The display name of the user. */
  name?: string;
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** Whether the user has verified their email address. */
  emailVerified: boolean;
}

export interface UserTenantPublic {
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** The display name of the user. */
  name?: string;
}

export interface UserLoginRequest {
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** The password of the user. */
  password: string;
}

export interface UserRegisterRequest {
  /** The name of the user. */
  name: string;
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** The password of the user. */
  password: string;
}

export interface UserTenantMembershipsList {
  pagination?: PaginationResponse;
  rows?: TenantMember[];
}

export interface Tenant {
  metadata: APIResourceMeta;
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
}

export interface TenantMember {
  metadata: APIResourceMeta;
  /** The user associated with this tenant member. */
  user: UserTenantPublic;
  /** The role of the user in the tenant. */
  role: TenantMemberRole;
  /** The tenant associated with this tenant member. */
  tenant?: Tenant;
}

export interface TenantMemberList {
  pagination?: PaginationResponse;
  rows?: TenantMember[];
}

export enum TenantMemberRole {
  OWNER = 'OWNER',
  ADMIN = 'ADMIN',
  MEMBER = 'MEMBER',
}

export interface CreateTenantInviteRequest {
  /** The email of the user to invite. */
  email: string;
  /** The role of the user in the tenant. */
  role: TenantMemberRole;
}

export interface UpdateTenantInviteRequest {
  /** The role of the user in the tenant. */
  role: TenantMemberRole;
}

export interface TenantInvite {
  metadata: APIResourceMeta;
  /** The email of the user to invite. */
  email: string;
  /** The role of the user in the tenant. */
  role: TenantMemberRole;
  /** The tenant id associated with this tenant invite. */
  tenantId: string;
  /** The tenant name for the tenant. */
  tenantName?: string;
  /**
   * The time that this invite expires.
   * @format date-time
   */
  expires: string;
}

export interface TenantInviteList {
  pagination?: PaginationResponse;
  rows?: TenantInvite[];
}

export interface AcceptInviteRequest {
  /**
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  invite: string;
}

export interface RejectInviteRequest {
  /**
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  invite: string;
}

export interface TenantList {
  pagination?: PaginationResponse;
  rows?: Tenant[];
}

export interface CreateTenantRequest {
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
}

export interface Event {
  metadata: APIResourceMeta;
  /** The key for the event. */
  key: string;
  /** The tenant associated with this event. */
  tenant?: Tenant;
  /** The ID of the tenant associated with this event. */
  tenantId: string;
  /** The workflow run summary for this event. */
  workflowRunSummary?: EventWorkflowRunSummary;
}

export interface EventData {
  /** The data for the event (JSON bytes). */
  data: string;
}

export interface EventWorkflowRunSummary {
  /**
   * The number of pending runs.
   * @format int64
   */
  pending?: number;
  /**
   * The number of running runs.
   * @format int64
   */
  running?: number;
  /**
   * The number of succeeded runs.
   * @format int64
   */
  succeeded?: number;
  /**
   * The number of failed runs.
   * @format int64
   */
  failed?: number;
}

export enum EventOrderByField {
  CreatedAt = 'createdAt',
}

export enum EventOrderByDirection {
  Asc = 'asc',
  Desc = 'desc',
}

export type EventSearch = string;

export interface EventKeyList {
  pagination?: PaginationResponse;
  rows?: EventKey[];
}

/** The key for the event. */
export type EventKey = string;

/** A workflow ID. */
export type WorkflowID = string;

export interface EventList {
  pagination?: PaginationResponse;
  rows?: Event[];
}

export interface ReplayEventRequest {
  eventIds: string[];
}

export interface Workflow {
  metadata: APIResourceMeta;
  /** The name of the workflow. */
  name: string;
  /** The description of the workflow. */
  description?: string;
  versions?: WorkflowVersionMeta[];
  /** The tags of the workflow. */
  tags?: WorkflowTag[];
  lastRun?: WorkflowRun;
  /** The jobs of the workflow. */
  jobs?: Job[];
  deployment?: WorkflowDeploymentConfig;
}

export interface WorkflowDeploymentConfig {
  metadata: APIResourceMeta;
  /** The repository name. */
  gitRepoName: string;
  /** The repository owner. */
  gitRepoOwner: string;
  /** The repository branch. */
  gitRepoBranch: string;
  /** The Github App installation. */
  githubAppInstallation?: GithubAppInstallation;
  /**
   * The id of the Github App installation.
   * @format uuid
   */
  githubAppInstallationId: string;
}

export interface WorkflowVersionMeta {
  metadata: APIResourceMeta;
  /** The version of the workflow. */
  version: string;
  /** @format int32 */
  order: number;
  workflowId: string;
  workflow?: Workflow;
}

export interface WorkflowVersion {
  metadata: APIResourceMeta;
  /** The version of the workflow. */
  version: string;
  /** @format int32 */
  order: number;
  workflowId: string;
  workflow?: Workflow;
  triggers?: WorkflowTriggers;
  jobs?: Job[];
}

export interface WorkflowVersionDefinition {
  /** The raw YAML definition of the workflow. */
  rawDefinition: string;
}

export interface WorkflowTag {
  /** The name of the workflow. */
  name: string;
  /** The description of the workflow. */
  color: string;
}

export interface WorkflowList {
  metadata?: APIResourceMeta;
  rows?: Workflow[];
  pagination?: PaginationResponse;
}

export interface WorkflowTriggers {
  metadata?: APIResourceMeta;
  workflow_version_id?: string;
  tenant_id?: string;
  events?: WorkflowTriggerEventRef[];
  crons?: WorkflowTriggerCronRef[];
}

export interface WorkflowTriggerEventRef {
  parent_id?: string;
  event_key?: string;
}

export interface WorkflowTriggerCronRef {
  parent_id?: string;
  cron?: string;
}

export interface Job {
  metadata: APIResourceMeta;
  tenantId: string;
  versionId: string;
  name: string;
  /** The description of the job. */
  description?: string;
  steps: Step[];
  /** The timeout of the job. */
  timeout?: string;
}

export interface Step {
  metadata: APIResourceMeta;
  /** The readable id of the step. */
  readableId: string;
  tenantId: string;
  jobId: string;
  action: string;
  /** The timeout of the step. */
  timeout?: string;
  children?: string[];
  parents?: string[];
}

export interface WorkflowRun {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowVersionId: string;
  workflowVersion?: WorkflowVersion;
  status: WorkflowRunStatus;
  displayName?: string;
  jobRuns?: JobRun[];
  triggeredBy: WorkflowRunTriggeredBy;
  input?: Record<string, any>;
  error?: string;
  /** @format date-time */
  startedAt?: string;
  /** @format date-time */
  finishedAt?: string;
}

export interface WorkflowRunList {
  rows?: WorkflowRun[];
  pagination?: PaginationResponse;
}

export enum WorkflowRunStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
}

export enum JobRunStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
}

export enum StepRunStatus {
  PENDING = 'PENDING',
  PENDING_ASSIGNMENT = 'PENDING_ASSIGNMENT',
  ASSIGNED = 'ASSIGNED',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
}

export interface JobRun {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowRunId: string;
  workflowRun?: WorkflowRun;
  jobId: string;
  job?: Job;
  tickerId?: string;
  stepRuns?: StepRun[];
  status: JobRunStatus;
  result?: object;
  /** @format date-time */
  startedAt?: string;
  /** @format date-time */
  finishedAt?: string;
  /** @format date-time */
  timeoutAt?: string;
  /** @format date-time */
  cancelledAt?: string;
  cancelledReason?: string;
  cancelledError?: string;
}

export interface WorkflowRunTriggeredBy {
  metadata: APIResourceMeta;
  parentId: string;
  eventId?: string;
  event?: Event;
  cronParentId?: string;
  cronSchedule?: string;
}

export interface StepRun {
  metadata: APIResourceMeta;
  tenantId: string;
  jobRunId: string;
  jobRun?: JobRun;
  stepId: string;
  step?: Step;
  children?: string[];
  parents?: string[];
  workerId?: string;
  input?: string;
  output?: string;
  status: StepRunStatus;
  /** @format date-time */
  requeueAfter?: string;
  result?: object;
  error?: string;
  /** @format date-time */
  startedAt?: string;
  startedAtEpoch?: number;
  /** @format date-time */
  finishedAt?: string;
  finishedAtEpoch?: number;
  /** @format date-time */
  timeoutAt?: string;
  timeoutAtEpoch?: number;
  /** @format date-time */
  cancelledAt?: string;
  cancelledAtEpoch?: number;
  cancelledReason?: string;
  cancelledError?: string;
  inputSchema?: string;
}

export interface WorkerList {
  pagination?: PaginationResponse;
  rows?: Worker[];
}

export interface Worker {
  metadata: APIResourceMeta;
  /** The name of the worker. */
  name: string;
  /**
   * The time this worker last sent a heartbeat.
   * @format date-time
   * @example "2022-12-13T20:06:48.888Z"
   */
  lastHeartbeatAt?: string;
  /** The actions this worker can perform. */
  actions?: string[];
  /** The recent step runs for this worker. */
  recentStepRuns?: StepRun[];
}

export interface APIToken {
  metadata: APIResourceMeta;
  /**
   * The name of the API token.
   * @maxLength 255
   */
  name: string;
  /**
   * When the API token expires.
   * @format date-time
   */
  expiresAt: string;
}

export interface CreateAPITokenRequest {
  /**
   * A name for the API token.
   * @maxLength 255
   */
  name: string;
}

export interface CreateAPITokenResponse {
  /** The API token. */
  token: string;
}

export interface ListAPITokensResponse {
  pagination?: PaginationResponse;
  rows?: APIToken[];
}

export interface RerunStepRunRequest {
  input: object;
}

export interface TriggerWorkflowRunRequest {
  input: object;
}

export interface LinkGithubRepositoryRequest {
  /**
   * The repository name.
   * @minLength 36
   * @maxLength 36
   */
  installationId: string;
  /** The repository name. */
  gitRepoName: string;
  /** The repository owner. */
  gitRepoOwner: string;
  /** The repository branch. */
  gitRepoBranch: string;
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

export interface CreatePullRequestFromStepRun {
  branchName: string;
}

export interface GetStepRunDiffResponse {
  diffs: StepRunDiff[];
}

export interface StepRunDiff {
  key: string;
  original: string;
  modified: string;
}

export interface ListPullRequestsResponse {
  pullRequests: PullRequest[];
}

export interface PullRequest {
  repositoryOwner: string;
  repositoryName: string;
  pullRequestID: number;
  pullRequestTitle: string;
  pullRequestNumber: number;
  pullRequestHeadBranch: string;
  pullRequestBaseBranch: string;
  pullRequestState: PullRequestState;
}

export enum PullRequestState {
  Open = 'open',
  Closed = 'closed',
}

export interface LogLine {
  /**
   * The creation date of the log line.
   * @format date-time
   */
  createdAt: string;
  /** The log message. */
  message: string;
  /** The log metadata. */
  metadata: object;
}

export enum LogLineLevel {
  DEBUG = 'DEBUG',
  INFO = 'INFO',
  WARN = 'WARN',
  ERROR = 'ERROR',
}

export interface LogLineList {
  pagination?: PaginationResponse;
  rows?: LogLine[];
}

export enum LogLineOrderByField {
  CreatedAt = 'createdAt',
}

export enum LogLineOrderByDirection {
  Asc = 'asc',
  Desc = 'desc',
}

export type LogLineSearch = string;

export type LogLineLevelField = LogLineLevel[];
