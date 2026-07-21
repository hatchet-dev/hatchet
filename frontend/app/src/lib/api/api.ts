import { inferControlPlaneEnabled } from './control-plane-status';
import {
  ExchangeTokenFetchError,
  exchangeTokenQueryOptions,
} from './exchange-token';
import { Api } from './generated/Api';
import { Api as CloudApi } from './generated/cloud/Api';
import { Api as ControlPlaneApi } from './generated/control-plane/Api';
import queryClient from '@/query-client';
import { InternalAxiosRequestConfig } from 'axios';
import qs from 'qs';
import { validate as validateUuid } from 'uuid';

type HttpErrorLike = {
  status?: unknown;
  response?: {
    status?: unknown;
  };
};

export function getApiErrorStatus(error: unknown): number | undefined {
  if (!error || typeof error !== 'object') {
    return undefined;
  }

  const maybeError = error as HttpErrorLike;
  const status = maybeError.status ?? maybeError.response?.status;

  return typeof status === 'number' ? status : undefined;
}

// Extend Axios config with custom fields injected by the API code generator.
// https://www.typescriptlang.org/docs/handbook/declaration-merging.html
declare module 'axios' {
  // AxiosRequestConfig is what callers pass in (and what RequestParams in the
  // generated http-client extends), so xTenantId must live here for it to be
  // accepted by api.tenantGet(id, { xTenantId: id }) without a cast.
  interface AxiosRequestConfig {
    xResources?: string[];
    // Explicitly identifies which tenant's exchange token should be used.
    // When set, the interceptor skips the localStorage fallback.
    xTenantId?: string;
    // Forces the exchange token interceptor to run even for non-tenant-scoped
    // endpoints (e.g. /api/v1/billing/plans).
    useExchangeToken?: boolean;
  }
  // InternalAxiosRequestConfig is what the interceptor receives after Axios
  // merges defaults, so the fields must be declared here too.
  interface InternalAxiosRequestConfig {
    xResources?: string[];
    xTenantId?: string;
    useExchangeToken?: boolean;
  }
}

const api = new Api({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

export default api;

export const cloudApi = new CloudApi({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

// Control-plane calls are all fast CRUD, but they cross regions for users on
// non-US shards; bound them well below the browser's own limit (~60s in
// Safari) so a stuck request fails into the retry path quickly.
const CONTROL_PLANE_REQUEST_TIMEOUT_MS = 15_000;

export const controlPlaneApi = new ControlPlaneApi({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
  timeout: CONTROL_PLANE_REQUEST_TIMEOUT_MS,
});

api.instance.interceptors.request.use(exchangeTokenInterceptor);
cloudApi.instance.interceptors.request.use(exchangeTokenInterceptor);

export const CONTROL_PLANE_TENANT_STORAGE_KEY = 'controlPlaneLastTenant';

type StoredTenantLike = {
  metadata?: {
    id?: string;
  };
};

function readStoredTenantId(): string | null {
  try {
    const raw = localStorage.getItem(CONTROL_PLANE_TENANT_STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const lastTenant: StoredTenantLike = JSON.parse(raw);
    return lastTenant.metadata?.id ?? null;
  } catch {
    return null;
  }
}

function readTenantIdFromPathname(pathname: string): string | null {
  const segments = pathname.split('/').filter(Boolean);
  const tenantSegmentIndex = segments.indexOf('tenants');
  if (tenantSegmentIndex === -1) {
    return null;
  }

  const tenantId = segments[tenantSegmentIndex + 1];
  if (!tenantId || !validateUuid(tenantId)) {
    return null;
  }

  return tenantId;
}

export function readTenantIdFromLocation(): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  return readTenantIdFromPathname(window.location.pathname);
}

export function resolveExchangeTokenTenantId(
  config: Pick<InternalAxiosRequestConfig, 'xTenantId'>,
): string | null {
  return config.xTenantId ?? readTenantIdFromLocation() ?? readStoredTenantId();
}

const CONTROL_PLANE_META_QUERY_KEY = ['control-plane-metadata:get'] as const;

type ControlPlaneMetaResponse = Awaited<
  ReturnType<typeof controlPlaneApi.metadataGet>
>;

/**
 * Shared query config for control-plane metadata.
 * Exported so loaders and the hook can reuse the same key/fn/staleTime,
 * keeping a single React Query cache entry as the source of truth.
 */
export const controlPlaneMetaQuery = {
  queryKey: CONTROL_PLANE_META_QUERY_KEY,
  queryFn: async (): Promise<ControlPlaneMetaResponse | null> => {
    try {
      return await controlPlaneApi.metadataGet();
    } catch (err) {
      // The control plane doesn't appear or disappear at runtime; once we
      // have an answer (including a definitive null), keep serving it so a
      // transient failure can't flip the app into "control plane disabled"
      // mode. That would make the exchange-token interceptor skip routing
      // tenant requests to the shard apiUrl, sending them to the
      // control-plane host where those routes don't exist (hard 404s).
      const lastKnown =
        queryClient.getQueryData<ControlPlaneMetaResponse | null>(
          CONTROL_PLANE_META_QUERY_KEY,
        );
      if (lastKnown !== undefined) {
        return lastKnown;
      }

      // First load: a 4xx (404 on OSS deployments without a control plane)
      // is a definitive "no control plane" answer.
      const status = getApiErrorStatus(err);
      if (status && status >= 400 && status < 500) {
        return null;
      }

      // First load with a transient error (network, timeout, 5xx): fail so
      // React Query retries, rather than caching "disabled" for staleTime.
      throw err;
    }
  },
  // Whether the control plane exists doesn't change at runtime, so a longer
  // staleTime just avoids blocking a request on a metadata round trip.
  staleTime: 1000 * 60 * 5,
};

/**
 * Fetches control-plane metadata (cached via React Query) and derives whether
 * the control plane is enabled. Use this in any async non-hook context (route
 * loaders, plain async functions) instead of calling the two steps separately.
 */
export async function fetchControlPlaneStatus() {
  const meta = await queryClient.fetchQuery(controlPlaneMetaQuery);
  return { meta, isControlPlaneEnabled: inferControlPlaneEnabled(meta?.data) };
}

async function resolveControlPlaneEnabled(): Promise<boolean> {
  return (await fetchControlPlaneStatus()).isControlPlaneEnabled;
}

/**
 * Exchange-token request interceptor.
 *
 * When control plane is active, tenant-scoped OSS API requests are
 * transparently authenticated with an exchange token and routed to the OSS apiUrl.
 *
 * If the exchange token cannot be obtained (e.g. network error), the request
 * is rejected with the token error. Proceeding unchanged would send the
 * request to the control-plane host, where OSS routes don't exist, and the
 * resulting 404 is both misleading and never retried by React Query.
 */
export async function exchangeTokenInterceptor(
  config: InternalAxiosRequestConfig,
) {
  const resources = config.xResources ?? [];
  if (!resources.includes('tenant') && !config.useExchangeToken) {
    return config;
  }

  const cpEnabled = await resolveControlPlaneEnabled();

  // xTenantId takes precedence — callers that intentionally target a tenant
  // other than the current page tenant set it explicitly. Otherwise the
  // interceptor uses the tenant encoded in window.location, then storage.
  const tenantId = resolveExchangeTokenTenantId(config);

  if (!cpEnabled) {
    return config;
  }
  if (!tenantId) {
    return config;
  }

  try {
    const exchangeToken = await queryClient.fetchQuery(
      exchangeTokenQueryOptions(tenantId, () =>
        controlPlaneApi.exchangeTokenCreate(tenantId).then((r) => r.data),
      ),
    );

    config.baseURL = exchangeToken.apiUrl;
    config.withCredentials = false;
    config.headers.set('Authorization', `Bearer ${exchangeToken.token}`);
    config.headers.set('X-Exchange-Token', 'true');
  } catch (err) {
    console.error('[exchange-token interceptor] failed to get token', {
      url: config.url,
      tenantId,
      err,
    });
    throw new ExchangeTokenFetchError(tenantId, err);
  }

  return config;
}
