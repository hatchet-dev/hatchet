import { inferControlPlaneEnabled } from './control-plane-status';
import { exchangeTokenQueryOptions } from './exchange-token';
import { Api } from './generated/Api';
import { Api as CloudApi } from './generated/cloud/Api';
import { Api as ControlPlaneApi } from './generated/control-plane/Api';
import queryClient from '@/query-client';
import { InternalAxiosRequestConfig } from 'axios';
import qs from 'qs';

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
    // endpoints (e.g. /api/v1/control-plane/billing/plans).
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

export const controlPlaneApi = new ControlPlaneApi({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
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

/**
 * Shared query config for control-plane metadata.
 * Exported so loaders and the hook can reuse the same key/fn/staleTime,
 * keeping a single React Query cache entry as the source of truth.
 */
export const controlPlaneMetaQuery = {
  queryKey: ['control-plane-metadata:get'] as const,
  queryFn: async () => {
    try {
      return await controlPlaneApi.metadataGet();
    } catch {
      return null;
    }
  },
  staleTime: 1000 * 60,
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
 * proceeds unchanged and will likely fail with a 401, which React Query will
 * surface to the UI normally.
 */
export async function exchangeTokenInterceptor(
  config: InternalAxiosRequestConfig,
) {
  const resources = config.xResources ?? [];
  if (!resources.includes('tenant') && !config.useExchangeToken) {
    return config;
  }

  const cpEnabled = await resolveControlPlaneEnabled();

  // xTenantId takes precedence — callers that know the tenant ID at request
  // time set it explicitly to avoid relying on the localStorage fallback. this prevents race
  // conditions where the interceptor checks localStorage before it's updated with the new tenant ID.
  const tenantId = config.xTenantId ?? readStoredTenantId();

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
  }

  return config;
}
