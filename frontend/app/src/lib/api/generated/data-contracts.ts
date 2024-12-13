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

export interface APIMetaAuth {
  /**
   * the supported types of authentication
   * @example ["basic","google"]
   */
  schemes?: string[];
}

export interface APIMetaPosthog {
  /**
   * the PostHog API key
   * @example "phk_1234567890abcdef"
   */
  apiKey?: string;
  /**
   * the PostHog API host
   * @example "https://posthog.example.com"
   */
  apiHost?: string;
}

export interface APIMeta {
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

export interface APIErrors {
  errors: APIError[];
}

export interface APIMetaIntegration {
  /**
   * the name of the integration
   * @example "github"
   */
  name: string;
  /** whether this integration is enabled on the instance */
  enabled: boolean;
}

export type ListAPIMetaIntegration = APIMetaIntegration[];

export interface UserLoginRequest {
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** The password of the user. */
  password: string;
}

export interface APIResourceMeta {
  /**
   * the id of this resource, in UUID format
   * @minLength 0
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  id: string;
  /**
   * the time that this resource was created
   * @format date-time
   * @example "2022-12-13T15:06:48.888358-05:00"
   */
  createdAt: string;
  /**
   * the time that this resource was last updated
   * @format date-time
   * @example "2022-12-13T15:06:48.888358-05:00"
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
  /** Whether the user has a password set. */
  hasPassword?: boolean;
  /** A hash of the user's email address for use with Pylon Support Chat */
  emailHash?: string;
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

export interface SNSIntegration {
  metadata: APIResourceMeta;
  /**
   * The unique identifier for the tenant that the SNS integration belongs to.
   * @format uuid
   */
  tenantId: string;
  /** The Amazon Resource Name (ARN) of the SNS topic. */
  topicArn: string;
  /** The URL to send SNS messages to. */
  ingestUrl?: string;
}

export interface ListSNSIntegrations {
  pagination: PaginationResponse;
  rows: SNSIntegration[];
}

export interface CreateSNSIntegrationRequest {
  /** The Amazon Resource Name (ARN) of the SNS topic. */
  topicArn: string;
}

export interface TenantAlertEmailGroup {
  metadata: APIResourceMeta;
  /** A list of emails for users */
  emails: string[];
}

export interface TenantAlertEmailGroupList {
  pagination?: PaginationResponse;
  rows?: TenantAlertEmailGroup[];
}

export interface CreateTenantAlertEmailGroupRequest {
  /** A list of emails for users */
  emails: string[];
}

export enum TenantResource {
  WORKER = 'WORKER',
  EVENT = 'EVENT',
  WORKFLOW_RUN = 'WORKFLOW_RUN',
  CRON = 'CRON',
  SCHEDULE = 'SCHEDULE',
}

export interface TenantResourceLimit {
  metadata: APIResourceMeta;
  /** The resource associated with this limit. */
  resource: TenantResource;
  /** The limit associated with this limit. */
  limitValue: number;
  /** The alarm value associated with this limit to warn of approaching limit value. */
  alarmValue?: number;
  /** The current value associated with this limit. */
  value: number;
  /** The meter window for the limit. (i.e. 1 day, 1 week, 1 month) */
  window?: string;
  /**
   * The last time the limit was refilled.
   * @format date-time
   */
  lastRefill?: string;
}

export interface TenantResourcePolicy {
  /** A list of resource limits for the tenant. */
  limits: TenantResourceLimit[];
}

export interface UpdateTenantAlertEmailGroupRequest {
  /** A list of emails for users */
  emails: string[];
}

export interface SlackWebhook {
  metadata: APIResourceMeta;
  /**
   * The unique identifier for the tenant that the SNS integration belongs to.
   * @format uuid
   */
  tenantId: string;
  /** The team name associated with this slack webhook. */
  teamName: string;
  /** The team id associated with this slack webhook. */
  teamId: string;
  /** The channel name associated with this slack webhook. */
  channelName: string;
  /** The channel id associated with this slack webhook. */
  channelId: string;
}

export interface ListSlackWebhooks {
  pagination: PaginationResponse;
  rows: SlackWebhook[];
}

export interface UserChangePasswordRequest {
  /** The password of the user. */
  password: string;
  /** The new password for the user. */
  newPassword: string;
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

export interface UserTenantPublic {
  /**
   * The email address of the user.
   * @format email
   */
  email: string;
  /** The display name of the user. */
  name?: string;
}

export enum TenantMemberRole {
  OWNER = 'OWNER',
  ADMIN = 'ADMIN',
  MEMBER = 'MEMBER',
}

export interface Tenant {
  metadata: APIResourceMeta;
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
  /** Whether the tenant has opted out of analytics. */
  analyticsOptOut?: boolean;
  /** Whether to alert tenant members. */
  alertMemberEmails?: boolean;
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

export interface UserTenantMembershipsList {
  pagination?: PaginationResponse;
  rows?: TenantMember[];
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

export interface CreateTenantRequest {
  /** The name of the tenant. */
  name: string;
  /** The slug of the tenant. */
  slug: string;
}

export interface UpdateTenantRequest {
  /** The name of the tenant. */
  name?: string;
  /** Whether the tenant has opted out of analytics. */
  analyticsOptOut?: boolean;
  /** Whether to alert tenant members. */
  alertMemberEmails?: boolean;
  /** Whether to send alerts when workflow runs fail. */
  enableWorkflowRunFailureAlerts?: boolean;
  /** Whether to enable alerts when tokens are approaching expiration. */
  enableExpiringTokenAlerts?: boolean;
  /** Whether to enable alerts when tenant resources are approaching limits. */
  enableTenantResourceLimitAlerts?: boolean;
  /** The max frequency at which to alert. */
  maxAlertingFrequency?: string;
}

export interface TenantAlertingSettings {
  metadata: APIResourceMeta;
  /** Whether to alert tenant members. */
  alertMemberEmails?: boolean;
  /** Whether to send alerts when workflow runs fail. */
  enableWorkflowRunFailureAlerts?: boolean;
  /** Whether to enable alerts when tokens are approaching expiration. */
  enableExpiringTokenAlerts?: boolean;
  /** Whether to enable alerts when tenant resources are approaching limits. */
  enableTenantResourceLimitAlerts?: boolean;
  /** The max frequency at which to alert. */
  maxAlertingFrequency: string;
  /**
   * The last time an alert was sent.
   * @format date-time
   */
  lastAlertedAt?: string;
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

export interface ListAPITokensResponse {
  pagination?: PaginationResponse;
  rows?: APIToken[];
}

export interface CreateAPITokenRequest {
  /**
   * A name for the API token.
   * @maxLength 255
   */
  name: string;
  /** The duration for which the token is valid. */
  expiresIn?: string;
}

export interface CreateAPITokenResponse {
  /** The API token. */
  token: string;
}

/** A workflow ID. */
export type WorkflowID = string;

export interface QueueMetrics {
  /** The number of items in the queue. */
  numQueued: number;
  /** The number of items running. */
  numRunning: number;
  /** The number of items pending. */
  numPending: number;
}

export interface TenantQueueMetrics {
  /** The total queue metrics. */
  total?: QueueMetrics;
  workflow?: Record<string, QueueMetrics>;
  queues?: Record<string, number>;
}

export interface TenantStepRunQueueMetrics {
  queues?: Record<string, number>;
}

/** The key for the event. */
export type EventKey = string;

export enum WorkflowRunStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
  QUEUED = 'QUEUED',
}

export type WorkflowRunStatusList = WorkflowRunStatus[];

export type EventSearch = string;

export enum EventOrderByField {
  CreatedAt = 'createdAt',
}

export enum EventOrderByDirection {
  Asc = 'asc',
  Desc = 'desc',
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
   * The number of queued runs.
   * @format int64
   */
  queued?: number;
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
  /** Additional metadata for the event. */
  additionalMetadata?: object;
}

export interface EventList {
  pagination?: PaginationResponse;
  rows?: Event[];
}

export interface CreateEventRequest {
  /** The key for the event. */
  key: string;
  /** The data for the event. */
  data: object;
  /** Additional metadata for the event. */
  additionalMetadata?: object;
}

export interface BulkCreateEventRequest {
  events: CreateEventRequest[];
}

export interface Events {
  metadata: APIResourceMeta;
  /** The events. */
  events: Event[];
}

export interface ReplayEventRequest {
  eventIds: string[];
}

export interface CancelEventRequest {
  eventIds: string[];
}

export enum RateLimitOrderByField {
  Key = 'key',
  Value = 'value',
  LimitValue = 'limitValue',
}

export enum RateLimitOrderByDirection {
  Asc = 'asc',
  Desc = 'desc',
}

export interface RateLimit {
  /** The key for the rate limit. */
  key: string;
  /** The ID of the tenant associated with this rate limit. */
  tenantId: string;
  /** The maximum number of requests allowed within the window. */
  limitValue: number;
  /** The current number of requests made within the window. */
  value: number;
  /** The window of time in which the limitValue is enforced. */
  window: string;
  /**
   * The last time the rate limit was refilled.
   * @format date-time
   * @example "2022-12-13T15:06:48.888358-05:00"
   */
  lastRefill: string;
}

export interface RateLimitList {
  pagination?: PaginationResponse;
  rows?: RateLimit[];
}

export interface TenantMemberList {
  pagination?: PaginationResponse;
  rows?: TenantMember[];
}

export interface EventData {
  /** The data for the event (JSON bytes). */
  data: string;
}

export interface EventKeyList {
  pagination?: PaginationResponse;
  rows?: EventKey[];
}

export interface Workflow {
  metadata: APIResourceMeta;
  /** The name of the workflow. */
  name: string;
  /** The description of the workflow. */
  description?: string;
  /** Whether the workflow is paused. */
  isPaused?: boolean;
  versions?: WorkflowVersionMeta[];
  /** The tags of the workflow. */
  tags?: WorkflowTag[];
  /** The jobs of the workflow. */
  jobs?: Job[];
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

export interface WorkflowTag {
  /** The name of the workflow. */
  name: string;
  /** The description of the workflow. */
  color: string;
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

export interface WorkflowList {
  metadata?: APIResourceMeta;
  rows?: Workflow[];
  pagination?: PaginationResponse;
}

export interface ScheduleWorkflowRunRequest {
  input: object;
  additionalMetadata: object;
  /** @format date-time */
  triggerAt: string;
}

export enum ScheduledWorkflowsMethod {
  DEFAULT = 'DEFAULT',
  API = 'API',
}

export interface ScheduledWorkflows {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowVersionId: string;
  workflowId: string;
  workflowName: string;
  /** @format date-time */
  triggerAt: string;
  input?: Record<string, any>;
  additionalMetadata?: Record<string, any>;
  /** @format date-time */
  workflowRunCreatedAt?: string;
  workflowRunName?: string;
  workflowRunStatus?: WorkflowRunStatus;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  workflowRunId?: string;
  method: ScheduledWorkflowsMethod;
}

export enum ScheduledWorkflowsOrderByField {
  TriggerAt = 'triggerAt',
  CreatedAt = 'createdAt',
}

export enum WorkflowRunOrderByDirection {
  ASC = 'ASC',
  DESC = 'DESC',
}

export enum ScheduledRunStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
  QUEUED = 'QUEUED',
  SCHEDULED = 'SCHEDULED',
}

export interface ScheduledWorkflowsList {
  rows?: ScheduledWorkflows[];
  pagination?: PaginationResponse;
}

export interface CreateCronWorkflowTriggerRequest {
  input: object;
  additionalMetadata: object;
  cronName: string;
  cronExpression: string;
}

export enum CronWorkflowsMethod {
  DEFAULT = 'DEFAULT',
  API = 'API',
}

export interface CronWorkflows {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowVersionId: string;
  workflowId: string;
  workflowName: string;
  cron: string;
  name?: string;
  input?: Record<string, any>;
  additionalMetadata?: Record<string, any>;
  enabled: boolean;
  method: CronWorkflowsMethod;
}

export enum CronWorkflowsOrderByField {
  Name = 'name',
  CreatedAt = 'createdAt',
}

export interface CronWorkflowsList {
  rows?: CronWorkflows[];
  pagination?: PaginationResponse;
}

export interface WorkflowRunsCancelRequest {
  workflowRunIds: string[];
}

export interface WorkflowUpdateRequest {
  /** Whether the workflow is paused. */
  isPaused?: boolean;
}

export enum ConcurrencyLimitStrategy {
  CANCEL_IN_PROGRESS = 'CANCEL_IN_PROGRESS',
  DROP_NEWEST = 'DROP_NEWEST',
  QUEUE_NEWEST = 'QUEUE_NEWEST',
  GROUP_ROUND_ROBIN = 'GROUP_ROUND_ROBIN',
}

export interface WorkflowConcurrency {
  /**
   * The maximum number of concurrent workflow runs.
   * @format int32
   */
  maxRuns: number;
  /** The strategy to use when the concurrency limit is reached. */
  limitStrategy: ConcurrencyLimitStrategy;
  /** An action which gets the concurrency group for the WorkflowRun. */
  getConcurrencyGroup: string;
}

export interface WorkflowTriggerEventRef {
  parent_id?: string;
  event_key?: string;
}

export interface WorkflowTriggerCronRef {
  parent_id?: string;
  cron?: string;
}

export interface WorkflowTriggers {
  metadata?: APIResourceMeta;
  workflow_version_id?: string;
  tenant_id?: string;
  events?: WorkflowTriggerEventRef[];
  crons?: WorkflowTriggerCronRef[];
}

export interface WorkflowVersion {
  metadata: APIResourceMeta;
  /** The version of the workflow. */
  version: string;
  /** @format int32 */
  order: number;
  workflowId: string;
  /** The sticky strategy of the workflow. */
  sticky?: string;
  /**
   * The default priority of the workflow.
   * @format int32
   */
  defaultPriority?: number;
  workflow?: Workflow;
  concurrency?: WorkflowConcurrency;
  triggers?: WorkflowTriggers;
  scheduleTimeout?: string;
  jobs?: Job[];
}

export interface TriggerWorkflowRunRequest {
  input: object;
  additionalMetadata?: object;
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
  /** @example 1000 */
  duration?: number;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  parentId?: string;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  parentStepRunId?: string;
  additionalMetadata?: Record<string, any>;
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

export enum StepRunStatus {
  PENDING = 'PENDING',
  PENDING_ASSIGNMENT = 'PENDING_ASSIGNMENT',
  ASSIGNED = 'ASSIGNED',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
  CANCELLING = 'CANCELLING',
}

export interface StepRun {
  metadata: APIResourceMeta;
  tenantId: string;
  jobRunId: string;
  jobRun?: JobRun;
  stepId: string;
  step?: Step;
  childWorkflowsCount?: number;
  parents?: string[];
  childWorkflowRuns?: string[];
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
}

export enum JobRunStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  SUCCEEDED = 'SUCCEEDED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
}

export interface WorkflowRunTriggeredBy {
  metadata: APIResourceMeta;
  parentWorkflowRunId?: string;
  eventId?: string;
  cronParentId?: string;
  cronSchedule?: string;
}

export interface WorkflowMetrics {
  /** The number of runs for a specific group key (passed via filter) */
  groupKeyRunsCount?: number;
  /** The total number of concurrency group keys. */
  groupKeyCount?: number;
}

export enum LogLineLevel {
  DEBUG = 'DEBUG',
  INFO = 'INFO',
  WARN = 'WARN',
  ERROR = 'ERROR',
}

export type LogLineLevelField = LogLineLevel[];

export type LogLineSearch = string;

export enum LogLineOrderByField {
  CreatedAt = 'createdAt',
}

export enum LogLineOrderByDirection {
  Asc = 'asc',
  Desc = 'desc',
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

export interface LogLineList {
  pagination?: PaginationResponse;
  rows?: LogLine[];
}

export enum StepRunEventReason {
  REQUEUED_NO_WORKER = 'REQUEUED_NO_WORKER',
  REQUEUED_RATE_LIMIT = 'REQUEUED_RATE_LIMIT',
  SCHEDULING_TIMED_OUT = 'SCHEDULING_TIMED_OUT',
  ASSIGNED = 'ASSIGNED',
  STARTED = 'STARTED',
  ACKNOWLEDGED = 'ACKNOWLEDGED',
  FINISHED = 'FINISHED',
  FAILED = 'FAILED',
  RETRYING = 'RETRYING',
  CANCELLED = 'CANCELLED',
  TIMEOUT_REFRESHED = 'TIMEOUT_REFRESHED',
  REASSIGNED = 'REASSIGNED',
  TIMED_OUT = 'TIMED_OUT',
  SLOT_RELEASED = 'SLOT_RELEASED',
  RETRIED_BY_USER = 'RETRIED_BY_USER',
  WORKFLOW_RUN_GROUP_KEY_SUCCEEDED = 'WORKFLOW_RUN_GROUP_KEY_SUCCEEDED',
  WORKFLOW_RUN_GROUP_KEY_FAILED = 'WORKFLOW_RUN_GROUP_KEY_FAILED',
}

export enum StepRunEventSeverity {
  INFO = 'INFO',
  WARNING = 'WARNING',
  CRITICAL = 'CRITICAL',
}

export interface StepRunEvent {
  id: number;
  /** @format date-time */
  timeFirstSeen: string;
  /** @format date-time */
  timeLastSeen: string;
  stepRunId?: string;
  workflowRunId?: string;
  reason: StepRunEventReason;
  severity: StepRunEventSeverity;
  message: string;
  count: number;
  data?: object;
}

export interface StepRunEventList {
  pagination?: PaginationResponse;
  rows?: StepRunEvent[];
}

export interface StepRunArchive {
  stepRunId: string;
  order: number;
  input?: string;
  output?: string;
  /** @format date-time */
  startedAt?: string;
  error?: string;
  retryCount: number;
  /** @format date-time */
  createdAt: string;
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
}

export interface StepRunArchiveList {
  pagination?: PaginationResponse;
  rows?: StepRunArchive[];
}

export interface WorkflowWorkersCount {
  freeSlotCount?: number;
  maxSlotCount?: number;
  workflowRunId?: string;
}

export enum WorkflowKind {
  FUNCTION = 'FUNCTION',
  DURABLE = 'DURABLE',
  DAG = 'DAG',
}

export type WorkflowKindList = WorkflowKind[];

export enum WorkflowRunOrderByField {
  CreatedAt = 'createdAt',
  StartedAt = 'startedAt',
  FinishedAt = 'finishedAt',
  Duration = 'duration',
}

export interface WorkflowRunList {
  rows?: WorkflowRun[];
  pagination?: PaginationResponse;
}

export interface ReplayWorkflowRunsRequest {
  /** @maxLength 500 */
  workflowRunIds: string[];
}

export interface ReplayWorkflowRunsResponse {
  workflowRuns: WorkflowRun[];
}

export interface WorkflowRunsMetricsCounts {
  PENDING?: number;
  RUNNING?: number;
  SUCCEEDED?: number;
  FAILED?: number;
  QUEUED?: number;
}

export interface WorkflowRunsMetrics {
  counts?: WorkflowRunsMetricsCounts;
}

export interface WorkflowRunShape {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowId?: string;
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
  /** @example 1000 */
  duration?: number;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  parentId?: string;
  /**
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  parentStepRunId?: string;
  additionalMetadata?: Record<string, any>;
}

export interface RerunStepRunRequest {
  input: object;
}

export enum WorkerType {
  SELFHOSTED = 'SELFHOSTED',
  MANAGED = 'MANAGED',
  WEBHOOK = 'WEBHOOK',
}

export interface SemaphoreSlots {
  /**
   * The step run id.
   * @format uuid
   */
  stepRunId: string;
  /** The action id. */
  actionId: string;
  /**
   * The time this slot was started.
   * @format date-time
   */
  startedAt?: string;
  /**
   * The time this slot will timeout.
   * @format date-time
   */
  timeoutAt?: string;
  /**
   * The workflow run id.
   * @format uuid
   */
  workflowRunId: string;
  status: StepRunStatus;
}

export interface RecentStepRuns {
  metadata: APIResourceMeta;
  /** The action id. */
  actionId: string;
  status: StepRunStatus;
  /** @format date-time */
  startedAt?: string;
  /** @format date-time */
  finishedAt?: string;
  /** @format date-time */
  cancelledAt?: string;
  /** @format uuid */
  workflowRunId: string;
}

export interface WorkerLabel {
  metadata: APIResourceMeta;
  /** The key of the label. */
  key: string;
  /** The value of the label. */
  value?: string;
}

export enum WorkerRuntimeSDKs {
  GOLANG = 'GOLANG',
  PYTHON = 'PYTHON',
  TYPESCRIPT = 'TYPESCRIPT',
}

export interface WorkerRuntimeInfo {
  sdkVersion?: string;
  language?: WorkerRuntimeSDKs;
  languageVersion?: string;
  os?: string;
  runtimeExtra?: string;
}

export interface Worker {
  metadata: APIResourceMeta;
  /** The name of the worker. */
  name: string;
  type: WorkerType;
  /**
   * The time this worker last sent a heartbeat.
   * @format date-time
   * @example "2022-12-13T15:06:48.888358-05:00"
   */
  lastHeartbeatAt?: string;
  /**
   * The time this worker last sent a heartbeat.
   * @format date-time
   * @example "2022-12-13T15:06:48.888358-05:00"
   */
  lastListenerEstablished?: string;
  /** The actions this worker can perform. */
  actions?: string[];
  /** The semaphore slot state for the worker. */
  slots?: SemaphoreSlots[];
  /** The recent step runs for the worker. */
  recentStepRuns?: RecentStepRuns[];
  /** The status of the worker. */
  status?: 'ACTIVE' | 'INACTIVE' | 'PAUSED';
  /** The maximum number of runs this worker can execute concurrently. */
  maxRuns?: number;
  /** The number of runs this worker can execute concurrently. */
  availableRuns?: number;
  /**
   * the id of the assigned dispatcher, in UUID format
   * @format uuid
   * @minLength 36
   * @maxLength 36
   * @example "bb214807-246e-43a5-a25d-41761d1cff9e"
   */
  dispatcherId?: string;
  /** The current label state of the worker. */
  labels?: WorkerLabel[];
  /** The webhook URL for the worker. */
  webhookUrl?: string;
  /**
   * The webhook ID for the worker.
   * @format uuid
   */
  webhookId?: string;
  runtimeInfo?: WorkerRuntimeInfo;
}

export interface WorkerList {
  pagination?: PaginationResponse;
  rows?: Worker[];
}

export interface UpdateWorkerRequest {
  /** Whether the worker is paused and cannot accept new runs. */
  isPaused?: boolean;
}

export interface WebhookWorker {
  metadata: APIResourceMeta;
  /** The name of the webhook worker. */
  name: string;
  /** The webhook url. */
  url: string;
}

export interface WebhookWorkerListResponse {
  pagination?: PaginationResponse;
  rows?: WebhookWorker[];
}

export interface WebhookWorkerCreateRequest {
  /** The name of the webhook worker. */
  name: string;
  /** The webhook url. */
  url: string;
  /**
   * The secret key for validation. If not provided, a random secret will be generated.
   * @minLength 32
   */
  secret?: string;
}

export interface WebhookWorkerCreated {
  metadata: APIResourceMeta;
  /** The name of the webhook worker. */
  name: string;
  /** The webhook url. */
  url: string;
  /** The secret key for validation. */
  secret: string;
}

export enum WebhookWorkerRequestMethod {
  GET = 'GET',
  POST = 'POST',
  PUT = 'PUT',
}

export interface WebhookWorkerRequest {
  /**
   * The date and time the request was created.
   * @format date-time
   */
  created_at: string;
  /** The HTTP method used for the request. */
  method: WebhookWorkerRequestMethod;
  /** The HTTP status code of the response. */
  statusCode: number;
}

export interface WebhookWorkerRequestListResponse {
  /** The list of webhook requests. */
  requests?: WebhookWorkerRequest[];
}

export interface TenantList {
  pagination?: PaginationResponse;
  rows?: Tenant[];
}

export interface WorkflowVersionDefinition {
  /** The raw YAML definition of the workflow. */
  rawDefinition: string;
}

export interface CreatePullRequestFromStepRun {
  branchName: string;
}

export interface StepRunDiff {
  key: string;
  original: string;
  modified: string;
}

export interface GetStepRunDiffResponse {
  diffs: StepRunDiff[];
}

export enum PullRequestState {
  Open = 'open',
  Closed = 'closed',
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

export interface ListPullRequestsResponse {
  pullRequests: PullRequest[];
}

export interface WebhookWorkerCreateResponse {
  worker?: WebhookWorkerCreated;
}

export type BulkCreateEventResponse = Events;
