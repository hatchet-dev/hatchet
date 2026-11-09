import { takeFragmentToken } from '../pages/auth/magic-link/magic-link';

function captureMagicLinkToken(): string | null {
  // Support deployments served under a base path and the router's trailing slash.
  if (!/\/auth\/magic-link\/?$/.test(window.location.pathname)) {
    return null;
  }

  const { token, urlWithoutFragment } = takeFragmentToken(window.location.href);
  if (window.location.hash) {
    window.history.replaceState(window.history.state, '', urlWithoutFragment);
  }
  return token;
}

function captureSupportRedeemCode(): string | null {
  if (!/\/support\/redeem\/?$/.test(window.location.pathname)) {
    return null;
  }

  const { token, urlWithoutFragment } = takeFragmentToken(window.location.href);
  if (window.location.hash) {
    window.history.replaceState(window.history.state, '', urlWithoutFragment);
  }
  return token;
}

// Keep the credentials in memory across lazy loading and React remounts.
export const initialMagicLinkToken = captureMagicLinkToken();
export const initialSupportRedeemCode = captureSupportRedeemCode();
