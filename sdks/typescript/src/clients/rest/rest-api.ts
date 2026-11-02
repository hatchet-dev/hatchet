/**
 * The REST API surface the feature clients are written against, so that one implementation of
 * each feature serves both generated clients: the axios client the Node client builds and the
 * fetch client the core client builds. The two are generated from the same spec and differ
 * only in the per-request options their last parameter takes and in the response object they
 * resolve with, so the surface is the fetch client's operations with that parameter widened
 * to what both accept and the response narrowed to its `data`.
 */
import type { Api as FetchApi } from './generated-fetch/Api';
import type { HttpClient } from './generated-fetch/http-client';

/**
 * Per-request options both generated clients accept: `signal` aborts the request, `headers`
 * adds request headers, `format` selects how the response body is read. Anything else is
 * passed through to the client in use.
 */
export interface RestRequestParams {
  signal?: AbortSignal;
  headers?: Record<string, string>;
  format?: 'json' | 'text';
}

/**
 * Whether a parameter is the generated client's per-request options: the one parameter whose
 * type carries a `cancelToken`. Index-signature types (`Record<string, any>`) have every key
 * and are left alone.
 */
type IsRequestParams<T> = string extends keyof T
  ? false
  : 'cancelToken' extends keyof T
    ? true
    : false;

type OperationArgs<A extends unknown[]> = {
  [I in keyof A]: IsRequestParams<NonNullable<A[I]>> extends true ? RestRequestParams : A[I];
};

type Operation<F> = F extends (...args: infer A) => Promise<{ data: infer D }>
  ? (...args: OperationArgs<A>) => Promise<{ data: D }>
  : never;

/** The API operations, without the HTTP client members the generated class inherits. */
type OperationName = Exclude<keyof FetchApi, keyof HttpClient>;

/** Every operation of the generated REST client, as either generated client provides it. */
export type RestApi = { [K in OperationName]: Operation<FetchApi[K]> };

/**
 * What a REST feature client is built from. The Node client and the core client both satisfy
 * it, so `new CronClient(hatchet)` works with either.
 */
export interface RestClientDeps {
  api: RestApi;
  tenantId: string;
  config?: { namespace?: string };
}
