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

import {
  APIError,
  APIErrors,
  CreateTenantRequest,
  EventData,
  EventKey,
  EventKeyList,
  EventList,
  EventOrderByDirection,
  EventOrderByField,
  ReplayEventRequest,
  Tenant,
  User,
  UserLoginRequest,
  UserRegisterRequest,
  UserTenantMembershipsList,
  Worker,
  WorkerList,
  Workflow,
  WorkflowList,
  WorkflowRun,
  WorkflowRunList,
  WorkflowVersion,
  WorkflowVersionDefinition,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<SecurityDataType = unknown> extends HttpClient<SecurityDataType> {
  /**
   * @description Logs in a user.
   *
   * @tags User
   * @name UserUpdateLogin
   * @summary Login user
   * @request POST:/api/v1/users/login
   */
  userUpdateLogin = (data: UserLoginRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/login`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Gets the current user
   *
   * @tags User
   * @name UserGetCurrent
   * @summary Get current user
   * @request GET:/api/v1/users/current
   * @secure
   */
  userGetCurrent = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/current`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Registers a user.
   *
   * @tags User
   * @name UserCreate
   * @summary Register user
   * @request POST:/api/v1/users/register
   */
  userCreate = (data: UserRegisterRequest, params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/register`,
      method: "POST",
      body: data,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Logs out a user.
   *
   * @tags User
   * @name UserUpdateLogout
   * @summary Logout user
   * @request POST:/api/v1/users/logout
   * @secure
   */
  userUpdateLogout = (params: RequestParams = {}) =>
    this.request<User, APIErrors>({
      path: `/api/v1/users/logout`,
      method: "POST",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all tenant memberships for the current user
   *
   * @tags User
   * @name TenantMembershipsList
   * @summary List tenant memberships
   * @request GET:/api/v1/users/memberships
   * @secure
   */
  tenantMembershipsList = (params: RequestParams = {}) =>
    this.request<UserTenantMembershipsList, APIErrors>({
      path: `/api/v1/users/memberships`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Creates a new tenant
   *
   * @tags Tenant
   * @name TenantCreate
   * @summary Create tenant
   * @request POST:/api/v1/tenants
   * @secure
   */
  tenantCreate = (data: CreateTenantRequest, params: RequestParams = {}) =>
    this.request<Tenant, APIErrors | APIError>({
      path: `/api/v1/tenants`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all events for a tenant.
   *
   * @tags Event
   * @name EventList
   * @summary List events
   * @request GET:/api/v1/tenants/{tenant}/events
   * @secure
   */
  eventList = (
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /** A list of keys to filter by */
      keys?: EventKey[];
      /** What to order by */
      orderByField?: EventOrderByField;
      /** The order direction */
      orderByDirection?: EventOrderByDirection;
    },
    params: RequestParams = {},
  ) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Replays a list of events.
   *
   * @tags Event
   * @name EventUpdateReplay
   * @summary Replay events
   * @request POST:/api/v1/tenants/{tenant}/events/replay
   * @secure
   */
  eventUpdateReplay = (tenant: string, data: ReplayEventRequest, params: RequestParams = {}) =>
    this.request<EventList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/replay`,
      method: "POST",
      body: data,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Get the data for an event.
   *
   * @tags Event
   * @name EventDataGet
   * @summary Get event data
   * @request GET:/api/v1/events/{event}/data
   * @secure
   */
  eventDataGet = (event: string, params: RequestParams = {}) =>
    this.request<EventData, APIErrors>({
      path: `/api/v1/events/${event}/data`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Lists all event keys for a tenant.
   *
   * @tags Event
   * @name EventKeyList
   * @summary List event keys
   * @request GET:/api/v1/tenants/{tenant}/events/keys
   * @secure
   */
  eventKeyList = (tenant: string, params: RequestParams = {}) =>
    this.request<EventKeyList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/events/keys`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all workflows for a tenant
   *
   * @tags Workflow
   * @name WorkflowList
   * @summary Get workflows
   * @request GET:/api/v1/tenants/{tenant}/workflows
   * @secure
   */
  workflowList = (tenant: string, params: RequestParams = {}) =>
    this.request<WorkflowList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a workflow for a tenant
   *
   * @tags Workflow
   * @name WorkflowGet
   * @summary Get workflow
   * @request GET:/api/v1/workflows/{workflow}
   * @secure
   */
  workflowGet = (workflow: string, params: RequestParams = {}) =>
    this.request<Workflow, APIErrors>({
      path: `/api/v1/workflows/${workflow}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a workflow version for a tenant
   *
   * @tags Workflow
   * @name WorkflowVersionGet
   * @summary Get workflow version
   * @request GET:/api/v1/workflows/{workflow}/versions
   * @secure
   */
  workflowVersionGet = (
    workflow: string,
    query?: {
      /**
       * The workflow version. If not supplied, the latest version is fetched.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      version?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowVersion, APIErrors>({
      path: `/api/v1/workflows/${workflow}/versions`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a workflow version definition for a tenant
   *
   * @tags Workflow
   * @name WorkflowVersionGetDefinition
   * @summary Get workflow version definition
   * @request GET:/api/v1/workflows/{workflow}/versions/definition
   * @secure
   */
  workflowVersionGetDefinition = (
    workflow: string,
    query?: {
      /**
       * The workflow version. If not supplied, the latest version is fetched.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      version?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowVersionDefinition, APIErrors>({
      path: `/api/v1/workflows/${workflow}/versions/definition`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all workflow runs for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunList
   * @summary Get workflow runs
   * @request GET:/api/v1/tenants/{tenant}/workflows/runs
   * @secure
   */
  workflowRunList = (
    tenant: string,
    query?: {
      /**
       * The number to skip
       * @format int64
       */
      offset?: number;
      /**
       * The number to limit by
       * @format int64
       */
      limit?: number;
      /**
       * The event id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      eventId?: string;
      /**
       * The workflow id to get runs for.
       * @format uuid
       * @minLength 36
       * @maxLength 36
       */
      workflowId?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<WorkflowRunList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflows/runs`,
      method: "GET",
      query: query,
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a workflow run for a tenant
   *
   * @tags Workflow
   * @name WorkflowRunGet
   * @summary Get workflow run
   * @request GET:/api/v1/tenants/{tenant}/workflow-runs/{workflow-run}
   * @secure
   */
  workflowRunGet = (tenant: string, workflowRun: string, params: RequestParams = {}) =>
    this.request<WorkflowRun, APIErrors>({
      path: `/api/v1/tenants/${tenant}/workflow-runs/${workflowRun}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get all workers for a tenant
   *
   * @tags Worker
   * @name WorkerList
   * @summary Get workers
   * @request GET:/api/v1/tenants/{tenant}/worker
   * @secure
   */
  workerList = (tenant: string, params: RequestParams = {}) =>
    this.request<WorkerList, APIErrors>({
      path: `/api/v1/tenants/${tenant}/worker`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * @description Get a worker
   *
   * @tags Worker
   * @name WorkerGet
   * @summary Get worker
   * @request GET:/api/v1/workers/{worker}
   * @secure
   */
  workerGet = (worker: string, params: RequestParams = {}) =>
    this.request<Worker, APIErrors>({
      path: `/api/v1/workers/${worker}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
}
