import { AuthLayout } from '../components/auth-layout';
import { leaveMfa, useMfa } from './use-mfa';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { useTenantDetails } from '@/hooks/use-tenant';
import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import { useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import { useEffect, useRef } from 'react';

export default function MFA() {
  const navigate = useNavigate();
  const { organizationId } = useTenantDetails();
  const { isLoaded: universeLoaded } = useUserUniverse();
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
    if (!state || state.pending || codes) {
      return;
    }
    if (organizationId) {
      void navigate({
        to: appRoutes.organizationUserRoute.to,
        params: { organization: organizationId },
        replace: true,
      });
      return;
    }
    // organizationId stays empty both while memberships load and when this
    // account has none. Only the second case should leave this page.
    if (!universeLoaded) {
      return;
    }
    void navigate({ to: appRoutes.authenticatedRoute.to, replace: true });
  }, [codes, navigate, organizationId, state, universeLoaded]);

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
  }, [codes, passkeyCeremony, recovery, run, state]);

  return (
    <div className="ph-no-capture ph-no-record h-full overflow-y-auto">
      <AuthLayout>
        <div className="flex flex-col gap-3 text-center lg:text-left">
          <div className="flex justify-center pb-3 lg:hidden">
            <HatchetLogo className="h-8 w-auto" />
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            Verify it&apos;s you
          </h2>
          <p className="text-sm text-muted-foreground">
            Use a passkey or authenticator app to finish logging in.
          </p>
        </div>
        {(error || status.error) && (
          <Alert variant="destructive">
            <AlertDescription>
              {error || 'Your login expired. Log in again.'}
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
            Log in again
          </Button>
        )}
        {codes && (
          <section className="grid gap-3 rounded border p-4">
            <h2 className="font-semibold">Save your recovery codes</h2>
            <p className="text-sm">
              Keep these somewhere safe outside Hatchet. Use them once each if
              you lose your other methods. You won&apos;t see them again.
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
                  {recovery ? 'Use a passkey or app' : 'Use a recovery code'}
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
              Back to login
            </Button>
          </div>
        )}
      </AuthLayout>
    </div>
  );
}
