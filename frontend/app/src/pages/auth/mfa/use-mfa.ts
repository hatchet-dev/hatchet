import { config } from '@/config';
import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import {
  mfaPost,
  mfaStatus,
  PasskeyCancelled,
  passkeyCeremony,
} from '@/lib/api/mfa';
import queryClient from '@/query-client';
import { useQuery } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export const leaveMfa = (path: string) =>
  window.location.assign(`${config.BASE_PATH.replace(/\/$/, '')}${path}`);

// The control plane returns a same-origin path. Strip a base path that this
// app will add again, and ignore anything that is not a single-slash path.
function hasControlOrSpace(value: string) {
  for (let i = 0; i < value.length; i++) {
    if (value.charCodeAt(i) <= 0x20) {
      return true;
    }
  }
  return false;
}

export function mfaLoginPath(next: string | undefined): string {
  if (
    !next ||
    !next.startsWith('/') ||
    next.startsWith('//') ||
    next.includes('\\') ||
    hasControlOrSpace(next)
  ) {
    return '/';
  }
  const base = config.BASE_PATH.replace(/\/$/, '');
  if (base && (next === base || next.startsWith(`${base}/`))) {
    const stripped = next.slice(base.length);
    return stripped.startsWith('/') ? stripped : '/';
  }
  return next;
}

export const EXPIRED_MESSAGE = "That code didn't work or has expired.";

// Codes come from the control plane. The message strings stay as a fallback
// for a control plane that has not started sending codes yet.
const rejectedCode = 'mfa_invalid';
const rejectedProof = 'invalid or expired MFA verification';

const friendlyCodes: Record<string, string> = {
  [rejectedCode]: EXPIRED_MESSAGE,
  mfa_rate_limited: 'Too many attempts. Try again in 10 minutes.',
  mfa_totp_locked:
    'Authenticator codes are locked. Use a passkey or a recovery code.',
  mfa_step_up_required: 'Verify a method you already use first.',
  mfa_login_expired: 'Your login expired. Log in again.',
  mfa_email_unverified: 'Verify your email before adding a method.',
};

const friendlyMessages: Record<string, string> = {
  [rejectedProof]: EXPIRED_MESSAGE,
  'too many MFA attempts; try again in ten minutes':
    friendlyCodes.mfa_rate_limited,
  'authenticator codes are locked; use a passkey or recovery code':
    friendlyCodes.mfa_totp_locked,
  'Verify an enrolled method before adding one':
    friendlyCodes.mfa_step_up_required,
  'Verify an enrolled method before removing one':
    friendlyCodes.mfa_step_up_required,
  'Verify an enrolled method before replacing recovery codes':
    friendlyCodes.mfa_step_up_required,
  'Please sign in again': friendlyCodes.mfa_login_expired,
  'Login expired. Please sign in again': friendlyCodes.mfa_login_expired,
  'Verify your email before setting up MFA': friendlyCodes.mfa_email_unverified,
};

export function useMfa() {
  const status = useQuery({
    queryKey: ['mfa:status'],
    queryFn: mfaStatus,
    retry: false,
    staleTime: 0,
  });
  const [code, setCode] = useState('');
  const [recovery, setRecovery] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [setup, setSetup] = useState<{
    secret: string;
    uri: string;
    qr: string;
  }>();
  const [codes, setCodes] = useState<string[]>();
  const [removeFactor, setRemoveFactor] = useState<string>();
  const state = status.data;

  async function run(
    action: () => Promise<{ recoveryCodes?: string[]; next?: string } | void>,
    login = false,
  ) {
    setBusy(true);
    setError('');
    try {
      const result = await action();
      setCode('');
      setSetup(undefined);
      if (result?.recoveryCodes) {
        setCodes(result.recoveryCodes);
        // Leave the recovery codes on screen. Dropping the rest of the cache
        // here makes the app shell replace this page with its loading state.
        await status.refetch();
        return 'recovery';
      }
      if (login) {
        // Login completion must discard cached exchange tokens before the app loads.
        queryClient.removeQueries({
          predicate: (q) => q.queryKey[0] !== 'mfa:status',
        });
        leaveMfa(mfaLoginPath(result?.next));
        return true;
      }
      // A factor proof on the settings page only needs a new shard token.
      // Wiping the rest of the cache replaces this page with the shell loader.
      queryClient.removeQueries({
        queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
      });
      await status.refetch();
      return true;
    } catch (err) {
      if (err instanceof PasskeyCancelled) {
        return false;
      }
      const data = isAxiosError(err) ? err.response?.data : undefined;
      const code = typeof data?.code === 'string' ? data.code : undefined;
      const message =
        typeof data?.message === 'string'
          ? data.message
          : err instanceof Error
            ? err.message
            : undefined;
      setError(
        (code && friendlyCodes[code]) ||
          (message && (friendlyMessages[message] ?? message)) ||
          "That didn't work. Try again.",
      );
      // A rejected proof commits consumption of the enrollment ceremony, so the
      // QR on screen cannot be submitted again. Other failures leave it in place.
      if (
        setup &&
        isAxiosError(err) &&
        (code === rejectedCode || message === rejectedProof)
      ) {
        setSetup(undefined);
      }
      return false;
    } finally {
      setBusy(false);
    }
  }

  const pending = state?.pending === true;
  const enrolled = !!state && (state.totp || state.passkeys.length > 0);

  return {
    status,
    state,
    code,
    setCode,
    recovery,
    setRecovery,
    busy,
    error,
    setup,
    setSetup,
    codes,
    setCodes,
    removeFactor,
    setRemoveFactor,
    run,
    pending,
    enrolled,
    passkeyCeremony,
    mfaPost,
    setBusy,
    setError,
  };
}
