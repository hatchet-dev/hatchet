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

export const EXPIRED_MESSAGE = "That code didn't work or has expired.";

// The control plane returns its internal error strings. Keyed on the exact text
// because the HTTP status alone doesn't separate these cases.
const friendlyMessages: Record<string, string> = {
  'invalid or expired MFA verification': EXPIRED_MESSAGE,
  'too many MFA attempts; try again in ten minutes':
    'Too many attempts. Try again in 10 minutes.',
  'Verify an enrolled method before adding one':
    'Verify a method you already use first.',
  'Verify an enrolled method before removing one':
    'Verify a method you already use first.',
  'Please sign in again': 'Your login expired. Log in again.',
  'Login expired. Please sign in again': 'Your login expired. Log in again.',
  'Verify your email before setting up MFA':
    'Verify your email before adding a method.',
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
    action: () => Promise<{ recoveryCodes?: string[] } | void>,
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
        leaveMfa('/');
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
      const message = isAxiosError(err)
        ? err.response?.data?.message
        : err instanceof Error
          ? err.message
          : undefined;
      setError(
        typeof message === 'string'
          ? (friendlyMessages[message] ?? message)
          : "That didn't work. Try again.",
      );
      if (setup) {
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
