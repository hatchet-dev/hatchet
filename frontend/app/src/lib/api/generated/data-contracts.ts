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

export enum TenantMemberRole {
  OWNER = "OWNER",
  ADMIN = "ADMIN",
  MEMBER = "MEMBER",
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
  CreatedAt = "createdAt",
}

export enum EventOrderByDirection {
  Asc = "asc",
  Desc = "desc",
}

export interface EventKeyList {
  pagination?: PaginationResponse;
  rows?: EventKey[];
}

/** The key for the event. */
export type EventKey = string;

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
  nextId: string;
}

export interface WorkflowRun {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowVersionId: string;
  workflowVersion?: WorkflowVersion;
  status: WorkflowRunStatus;
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
  PENDING = "PENDING",
  RUNNING = "RUNNING",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELLED = "CANCELLED",
}

export enum JobRunStatus {
  PENDING = "PENDING",
  RUNNING = "RUNNING",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELLED = "CANCELLED",
}

export enum StepRunStatus {
  PENDING = "PENDING",
  PENDING_ASSIGNMENT = "PENDING_ASSIGNMENT",
  ASSIGNED = "ASSIGNED",
  RUNNING = "RUNNING",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELLED = "CANCELLED",
}

export interface JobRun {
  metadata: APIResourceMeta;
  tenantId: string;
  workflowRunId: string;
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
  stepId: string;
  step?: Step;
  nextId?: string;
  prevId?: string;
  workerId?: string;
  status: StepRunStatus;
  input?: object;
  /** @format date-time */
  requeueAfter?: string;
  result?: object;
  error?: string;
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
