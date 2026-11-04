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

export function accountSwitchMessage(
  currentEmail: string | null | undefined,
): string | null {
  if (!currentEmail) {
    return null;
  }

  return `You're signed in as ${currentEmail}. Continuing replaces this browser session with the account that received the login link.`;
}
