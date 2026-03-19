import {
  inferControlPlaneEnabled,
  readStoredControlPlaneEnabled,
  writeStoredControlPlaneEnabled,
} from './control-plane-status';
import { exchangeTokenQueryOptions } from './exchange-token';
import { Api } from './generated/Api';
import { Api as CloudApi } from './generated/cloud/Api';
import { Api as ControlPlaneApi } from './generated/control-plane/Api';
import queryClient from '@/query-client';
import qs from 'qs';

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

const LAST_TENANT_STORAGE_KEY = 'lastTenant';

type StoredTenantLike = {
  metadata?: {
    id?: string;
  };
};

function readStoredTenantId(): string | null {
  try {
    const raw = localStorage.getItem(LAST_TENANT_STORAGE_KEY);
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
 * Resolves whether control plane is enabled, preferring localStorage when
 * available. If unknown in localStorage, wait for metadata query resolution.
 */
async function resolveControlPlaneEnabled(): Promise<boolean> {
  const stored = readStoredControlPlaneEnabled();
  if (stored !== null) {
    return stored;
  }

  const cpMeta = await queryClient.fetchQuery({
    queryKey: ['control-plane-metadata:get'],
    queryFn: async () => {
      try {
        return await controlPlaneApi.metadataGet();
      } catch {
        return null;
      }
    },
    staleTime: 1000 * 60,
  });

  if (cpMeta === null) {
    return false;
  }

  const enabled = inferControlPlaneEnabled(cpMeta.data);
  writeStoredControlPlaneEnabled(enabled);
  return enabled;
}

/**
 * Exchange-token request interceptor.
 *
 * When control plane is active, tenant-scoped OSS API requests are
 * transparently authenticated with an exchange token:
 *
 *  - Tenant is resolved from persisted app state (`lastTenant`) instead of URL
 *    parsing, so endpoints that omit tenant in path still work.
 *  - Control-plane status is resolved from localStorage; if unknown, this
 *    interceptor waits for metadata to resolve before deciding.
 *  - A valid exchange token is obtained via `queryClient.fetchQuery`, which
 *    deduplicates concurrent fetches for the same tenant, caches the result
 *    in React Query memory (staleTime = token lifetime − 1 min), and reads
 *    from localStorage on startup to avoid an unnecessary round-trip.
 *  - The request's baseURL is rewritten to the tenant-specific API URL that
 *    is embedded in the exchange token claims.
 *  - `Authorization: Bearer <token>` and `X-Exchange-Token: true` headers
 *    are added so the OSS backend can identify and validate the token.
 *
 * If the exchange token cannot be obtained (e.g. network error), the request
 * proceeds unchanged and will likely fail with a 401, which React Query will
 * surface to the UI normally.
 */
api.instance.interceptors.request.use(async (config) => {
  const cpEnabled = await resolveControlPlaneEnabled();
  const tenantId = readStoredTenantId();

  console.log('[exchange-token interceptor]', {
    url: config.url,
    method: config.method,
    cpEnabled,
    tenantId: tenantId ?? '(none)',
  });

  if (!cpEnabled) return config;
  if (!tenantId) return config;

  try {
    // TODO-CONTROL-PLANE: it doesn't seem like this is using the cached token?
    const exchangeToken = await queryClient.fetchQuery(
      exchangeTokenQueryOptions(tenantId, () =>
        controlPlaneApi.exchangeTokenCreate(tenantId).then((r) => r.data),
      ),
    );

    console.log('[exchange-token interceptor] applying token', {
      url: config.url,
      rawApiUrl: exchangeToken.apiUrl,
      apiUrl: exchangeToken.apiUrl,
      expiresAt: exchangeToken.expiresAt,
    });

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
});
