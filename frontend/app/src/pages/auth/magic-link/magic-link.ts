export function showsMagicLinkLogin(
  isControlPlaneEnabled: boolean,
  schemes: readonly string[] | undefined,
): boolean {
  return isControlPlaneEnabled && (schemes ?? []).includes('magic_link');
}

export function takeFragmentToken(href: string): {
  token: string | null;
  urlWithoutFragment: string;
} {
  const url = new URL(href);
  let token: string | null = null;
  if (url.hash.length > 1) {
    try {
      token = decodeURIComponent(url.hash.slice(1));
    } catch {
      token = null;
    }
  }

  return {
    token: token || null,
    urlWithoutFragment: `${url.pathname}${url.search}`,
  };
}

// The confirm endpoint answers 400 when the token is invalid, expired, used, or
// not allowed, and 405 when magic links are disabled. Anything else (network,
// rate limit, server error) may succeed on retry.
export function isTerminalMagicLinkError(status: number | undefined): boolean {
  return status === 400 || status === 405;
}

export function accountSwitchMessage(
  currentEmail: string | null | undefined,
): string | null {
  if (!currentEmail) {
    return null;
  }

  return `You're signed in as ${currentEmail}. Continuing replaces this browser session with the account that received the login link.`;
}
