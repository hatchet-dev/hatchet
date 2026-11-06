import { config } from '@/config';
import { mfaPost, mfaStatus, passkeyCeremony } from '@/lib/api/mfa';
import queryClient from '@/query-client';
import { useQuery } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export const leaveMfa = (path: string) =>
  window.location.assign(`${config.BASE_PATH.replace(/\/$/, '')}${path}`);

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
      setRemoveFactor(undefined);
      if (result?.recoveryCodes) {
        setCodes(result.recoveryCodes);
      }
      // Factor changes and login completion must discard cached exchange tokens.
      queryClient.removeQueries({
        predicate: (q) => q.queryKey[0] !== 'mfa:status',
      });
      if (login) {
        leaveMfa('/');
        return;
      }
      await status.refetch();
    } catch (err) {
      const message = isAxiosError(err)
        ? err.response?.data?.message
        : err instanceof Error
          ? err.message
          : undefined;
      setError(
        typeof message === 'string'
          ? message
          : 'Verification failed. Please try again.',
      );
      if (setup) {
        setSetup(undefined);
      }
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
