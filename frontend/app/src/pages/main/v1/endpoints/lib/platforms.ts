import { CopyKey } from './copy-keys';
import { V1ServerlessEndpointKind } from '@/lib/api/generated/data-contracts';

export type PlatformInfo = {
  kind: V1ServerlessEndpointKind;
  // an example of the URL a user deploys to, from which the endpoint URLs are derived
  baseUrlPlaceholder?: string;
  // paths the @hatchet-dev/serverless adapter for the platform serves
  healthcheckPath?: string;
  triggerPath?: string;
  defaultRequestTimeoutSeconds: number;
  logsCommand?: string;
};

export const PLATFORMS: Record<V1ServerlessEndpointKind, PlatformInfo> = {
  [V1ServerlessEndpointKind.CLOUDFLARE_WORKERS]: {
    kind: V1ServerlessEndpointKind.CLOUDFLARE_WORKERS,
    baseUrlPlaceholder: 'https://my-worker.my-subdomain.workers.dev',
    healthcheckPath: '/hatchet/healthcheck',
    triggerPath: '/hatchet/trigger',
    defaultRequestTimeoutSeconds: 300,
    logsCommand: 'wrangler tail',
  },
  [V1ServerlessEndpointKind.VERCEL]: {
    kind: V1ServerlessEndpointKind.VERCEL,
    baseUrlPlaceholder: 'https://my-app.vercel.app',
    healthcheckPath: '/api/hatchet/healthcheck',
    triggerPath: '/api/hatchet/trigger',
    defaultRequestTimeoutSeconds: 300,
    logsCommand: 'vercel logs',
  },
  [V1ServerlessEndpointKind.GENERIC_HTTP]: {
    kind: V1ServerlessEndpointKind.GENERIC_HTTP,
    defaultRequestTimeoutSeconds: 300,
  },
};

export const PLATFORM_ORDER: V1ServerlessEndpointKind[] = [
  V1ServerlessEndpointKind.CLOUDFLARE_WORKERS,
  V1ServerlessEndpointKind.VERCEL,
  V1ServerlessEndpointKind.GENERIC_HTTP,
];

/**
 * Derives the healthcheck and trigger URLs from a deployment's base URL with the
 * paths the platform adapter serves. Returns empty strings for a platform
 * without fixed paths.
 */
export function deriveEndpointUrls(
  kind: V1ServerlessEndpointKind,
  baseUrl: string,
): { healthcheckUrl: string; triggerUrl: string } {
  const platform = PLATFORMS[kind];
  const base = baseUrl.trim().replace(/\/+$/, '');

  if (!base || !platform.healthcheckPath || !platform.triggerPath) {
    return { healthcheckUrl: '', triggerUrl: '' };
  }

  return {
    healthcheckUrl: `${base}${platform.healthcheckPath}`,
    triggerUrl: `${base}${platform.triggerPath}`,
  };
}

/**
 * Mirrors the API's registration check (safeclient.ValidateEndpoint): https,
 * a host, no credentials, and no explicit port other than 443. Returns the copy
 * key of the problem, or undefined when the URL is acceptable.
 */
export function validateEndpointUrl(raw: string): CopyKey | undefined {
  let url: URL;

  try {
    url = new URL(raw.trim());
  } catch {
    return 'validation.url.invalid';
  }

  if (url.protocol !== 'https:') {
    return 'validation.url.https';
  }

  if (!url.hostname) {
    return 'validation.url.host';
  }

  if (url.username || url.password) {
    return 'validation.url.credentials';
  }

  // URL drops the default port, so any port left is a non-443 one
  if (url.port && url.port !== '443') {
    return 'validation.url.port';
  }

  return undefined;
}

export const MIN_SIGNING_SECRET_LENGTH = 32;

/** A 64 character hex secret, the same shape as `openssl rand -hex 32`. */
export function generateSigningSecret(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);

  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}
