const SKIP_PREFIX = 'hatchet:domain-redirect-skip:';

export function domainRedirectSkipStorageKey(targetOrigin: string): string {
  return `${SKIP_PREFIX}${targetOrigin}`;
}

export function parseRedirectFrontendOrigin(raw: string): URL | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  try {
    if (trimmed.includes('://')) {
      return new URL(trimmed);
    }
    return new URL(`https://${trimmed}`);
  } catch {
    return null;
  }
}

export function buildRedirectFrontendHref(
  targetOrigin: URL,
  loc: Pick<Location, 'pathname' | 'search' | 'hash'>,
): string {
  return new URL(`${loc.pathname}${loc.search}${loc.hash}`, targetOrigin.origin)
    .href;
}
