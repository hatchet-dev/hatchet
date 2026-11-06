import { AuthLayout } from '../components/auth-layout';
import { leaveMfa, useMfa } from './use-mfa';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { useTenantDetails } from '@/hooks/use-tenant';
import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import { useEffect, useRef } from 'react';

export default function MFA() {
  const navigate = useNavigate();
  const { organizationId } = useTenantDetails();
  const {
    status,
    state,
    code,
    setCode,
    recovery,
    setRecovery,
    busy,
    error,
    codes,
    setCodes,
    run,
    pending,
    enrolled,
    passkeyCeremony,
    mfaPost,
  } = useMfa();

  const prompted = useRef(false);

  useEffect(() => {
    if (!state || state.pending || codes || !organizationId) {
      return;
    }
    void navigate({
      to: appRoutes.organizationUserRoute.to,
      params: { organization: organizationId },
      replace: true,
    });
  }, [codes, navigate, organizationId, state]);

  useEffect(() => {
    if (
      prompted.current ||
      !state?.pending ||
      codes ||
      recovery ||
      state.passkeys.length === 0
    ) {
      return;
    }
    prompted.current = true;
    void run(() => passkeyCeremony(false), true);
  }, [codes, recovery, run, state]);

  return (
    <div className="ph-no-capture ph-no-record h-full overflow-y-auto">
      <AuthLayout>
        <h1 className="text-2xl font-semibold">
          {pending ? 'Verify your sign-in' : 'Account security'}
        </h1>
        <p className="text-sm text-muted-foreground">
          {pending
            ? 'Use a passkey, authenticator code, or recovery code to finish signing in.'
            : 'Account security is managed in Settings.'}
        </p>
        {(error || status.error) && (
          <Alert variant="destructive">
            <AlertDescription>
              {error || 'Sign in again to manage or verify your MFA.'}
            </AlertDescription>
          </Alert>
        )}
        {status.isPending && <p>Loading…</p>}
        {status.isError && (
          <Button
            onClick={() =>
              run(async () => {
                await mfaPost('cancel');
                leaveMfa('/auth/login');
              })
            }
          >
            Sign in again
          </Button>
        )}
        {codes && (
          <section className="grid gap-3 rounded border p-4">
            <h2 className="font-semibold">Save your recovery codes</h2>
            <p className="text-sm">
              Keep these somewhere safe outside Hatchet. Each code works once,
              after your normal login. They will not be shown again.
            </p>
            <pre className="overflow-auto text-xs select-all">
              {codes.join('\n')}
            </pre>
            <Button
              onClick={() => {
                const blob = new Blob([codes.join('\n') + '\n'], {
                  type: 'text/plain',
                });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'hatchet-recovery-codes.txt';
                a.click();
                URL.revokeObjectURL(url);
              }}
            >
              Download codes
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                setCodes(undefined);
                queryClient.removeQueries({
                  queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
                });
              }}
            >
              I have saved these codes
            </Button>
          </section>
        )}
        {state && pending && !codes && (
          <div className="grid gap-4">
            {enrolled && (
              <section className="grid gap-3">
                {state.passkeys.length > 0 && (
                  <Button
                    disabled={busy}
                    onClick={() => run(() => passkeyCeremony(false), true)}
                  >
                    Verify with a passkey
                  </Button>
                )}
                {(state.totp || recovery) && (
                  <form
                    className="grid gap-2"
                    onSubmit={(event) => {
                      event.preventDefault();
                      void run(
                        () =>
                          mfaPost(`${recovery ? 'recovery' : 'totp'}/finish`, {
                            code: code.trim(),
                          }),
                        true,
                      );
                    }}
                  >
                    <Label htmlFor="mfa-code">
                      {recovery ? 'Recovery code' : 'Authenticator code'}
                    </Label>
                    <Input
                      id="mfa-code"
                      value={code}
                      onChange={(e) => setCode(e.target.value)}
                      autoComplete="one-time-code"
                      inputMode={recovery ? 'text' : 'numeric'}
                      maxLength={recovery ? 64 : 6}
                      disabled={busy}
                    />
                    <Button disabled={busy || !code.trim()}>Verify code</Button>
                  </form>
                )}
                <Button
                  variant="ghost"
                  disabled={busy}
                  onClick={() => {
                    setRecovery(!recovery);
                    setCode('');
                  }}
                >
                  {recovery
                    ? 'Use an authenticator or passkey'
                    : 'Use a recovery code'}
                </Button>
              </section>
            )}
            <Button
              variant="ghost"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await mfaPost('cancel');
                  leaveMfa('/auth/login');
                })
              }
            >
              Cancel and sign in again
            </Button>
          </div>
        )}
      </AuthLayout>
    </div>
  );
}
