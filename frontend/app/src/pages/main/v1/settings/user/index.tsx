import { SettingsPageHeader } from '../../tenant-settings/components/settings-page-header';
import { SettingRow } from '../../tenant-settings/components/settings-row';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { leaveMfa, useMfa } from '@/pages/auth/mfa/use-mfa';

export default function UserSecuritySettings() {
  const mfa = useMfa();
  const {
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
    enrolled,
    passkeyCeremony,
    mfaPost,
    setBusy,
    setError,
  } = mfa;

  return (
    <div className="ph-no-capture ph-no-record h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Security"
          description="Add an optional second check to every Cloud login, including SSO and email links."
        />

        {(error || status.error) && (
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>
              {error || 'Sign in again to manage or verify your MFA.'}
            </AlertDescription>
          </Alert>
        )}
        {status.isPending && <Spinner />}
        {status.isError && (
          <Button
            variant="outline"
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
          <section className="mb-6 grid gap-3 rounded-md border p-4">
            <h2 className="text-sm font-medium">Save your recovery codes</h2>
            <p className="max-w-xl text-sm text-muted-foreground">
              Keep these somewhere safe outside Hatchet. Each code works once,
              after your normal login. They will not be shown again.
            </p>
            <pre className="overflow-auto text-xs select-all">
              {codes.join('\n')}
            </pre>
            <div className="flex flex-wrap gap-2">
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
              <Button variant="outline" onClick={() => setCodes(undefined)}>
                I have saved these codes
              </Button>
            </div>
          </section>
        )}
        {state && !codes && (
          <div className="divide-y divide-border">
            {!state.canManage && (
              <SettingRow
                label="Recent sign-in required"
                description="To change factors, sign in again within the last five minutes and verify an existing factor."
              >
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() =>
                    run(async () => {
                      await mfaPost('cancel');
                      leaveMfa('/auth/login');
                    })
                  }
                >
                  Sign out and sign in again
                </Button>
              </SettingRow>
            )}
            {enrolled && !state.canManage && (
              <SettingRow
                label={recovery ? 'Recovery code' : 'Verify this sign-in'}
                description="Confirm an existing factor before changing account security."
              >
                <div className="grid justify-items-end gap-2">
                  {state.passkeys.length > 0 && !recovery && (
                    <Button
                      disabled={busy}
                      onClick={() => run(() => passkeyCeremony(false))}
                    >
                      Verify with a passkey
                    </Button>
                  )}
                  {(state.totp || recovery) && (
                    <form
                      className="grid justify-items-end gap-2"
                      onSubmit={(event) => {
                        event.preventDefault();
                        void run(() =>
                          mfaPost(`${recovery ? 'recovery' : 'totp'}/finish`, {
                            code: code.trim(),
                          }),
                        );
                      }}
                    >
                      <Label htmlFor="mfa-code" className="sr-only">
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
                        className="w-56"
                      />
                      <Button disabled={busy || !code.trim()}>
                        Verify code
                      </Button>
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
                </div>
              </SettingRow>
            )}
            <SettingRow
              label="Authenticator"
              description={
                state.totp
                  ? 'An authenticator app is enabled for this account.'
                  : 'Use a time-based code from an authenticator app.'
              }
            >
              {!state.totp && !setup && (
                <Button
                  disabled={busy || !state.canManage}
                  onClick={async () => {
                    setBusy(true);
                    setError('');
                    try {
                      setSetup(await mfaPost('totp-enroll/start'));
                    } catch {
                      setError(
                        'Could not begin setup. Sign in again and retry.',
                      );
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  Set up an authenticator
                </Button>
              )}
              {state.totp && (
                <Button
                  variant="outline"
                  disabled={busy || !state.canManage}
                  onClick={() => setRemoveFactor('totp')}
                >
                  Remove authenticator
                </Button>
              )}
            </SettingRow>
            {setup && (
              <form
                className="grid max-w-md gap-3 py-4"
                onSubmit={(event) => {
                  event.preventDefault();
                  void run(() =>
                    mfaPost('totp-enroll/finish', { code: code.trim() }),
                  );
                }}
              >
                <p className="text-sm text-muted-foreground">
                  Scan this code in your authenticator app, or enter the setup
                  key. Then enter a code to enable MFA.
                </p>
                <img
                  src={setup.qr}
                  width={256}
                  height={256}
                  alt="Authenticator setup QR code"
                />
                <code className="break-all select-all text-xs">
                  {setup.secret}
                </code>
                <Label htmlFor="setup-code">Authenticator code</Label>
                <Input
                  id="setup-code"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  maxLength={6}
                />
                <div className="flex flex-wrap gap-2">
                  <Button disabled={busy || code.length !== 6}>
                    Confirm authenticator
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => {
                      setSetup(undefined);
                      setCode('');
                    }}
                  >
                    Cancel setup
                  </Button>
                </div>
              </form>
            )}
            <SettingRow
              label="Passkeys"
              description={
                state.passkeys.length > 0
                  ? `${state.passkeys.length} passkey(s) registered for this account.`
                  : 'Use a passkey as a second factor. Passkeys do not replace your password or SSO login.'
              }
            >
              <div className="flex flex-wrap justify-end gap-2">
                <Button
                  disabled={
                    busy || !state.canManage || state.passkeys.length >= 10
                  }
                  onClick={() => run(() => passkeyCeremony(true))}
                >
                  Add a passkey
                </Button>
                {state.passkeys.map((key, index) => (
                  <Button
                    key={key}
                    variant="outline"
                    disabled={busy || !state.canManage}
                    onClick={() => setRemoveFactor(key)}
                  >
                    Remove passkey {index + 1}
                  </Button>
                ))}
              </div>
            </SettingRow>
            {enrolled && (
              <SettingRow
                label="Recovery codes"
                description={`${state.recoveryRemaining} codes left. Each code works once, after your normal login.`}
              >
                <Button
                  variant="outline"
                  disabled={busy || !state.canManage}
                  onClick={() => run(() => mfaPost('recovery'))}
                >
                  Replace recovery codes
                </Button>
              </SettingRow>
            )}
            {removeFactor && (
              <Alert className="mt-4">
                <AlertDescription className="grid gap-2">
                  Remove this factor? Removing your last factor turns MFA off.
                  Other sessions will require a new login.
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="destructive"
                      disabled={busy}
                      onClick={() =>
                        run(() => mfaPost('remove', { factor: removeFactor }))
                      }
                    >
                      Confirm removal
                    </Button>
                    <Button
                      variant="ghost"
                      onClick={() => setRemoveFactor(undefined)}
                    >
                      Keep factor
                    </Button>
                  </div>
                </AlertDescription>
              </Alert>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
