import { SettingsPageHeader } from '../../tenant-settings/components/settings-page-header';
import { SettingRow } from '../../tenant-settings/components/settings-row';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import { type MFAMethod } from '@/lib/api/mfa';
import queryClient from '@/query-client';
import { leaveMfa, useMfa } from '@/pages/auth/mfa/use-mfa';
import { type FormEvent, useState } from 'react';

const methodLimit = 10;

const methodKindLabel: Record<MFAMethod['kind'], string> = {
  totp: 'Authenticator app',
  passkey: 'Passkey',
};

const methodTypeLabel: Record<MFAMethod['kind'], string> = {
  totp: 'Authenticator',
  passkey: 'Passkey',
};

// Authenticator apps show setup keys in groups of four, so a typed key is
// easier to check against the app.
function groupSecret(secret: string) {
  return secret.match(/.{1,4}/g)?.join(' ') ?? secret;
}

function methodName(method: MFAMethod, methods: MFAMethod[]) {
  const sameKind = methods.filter((item) => item.kind === method.kind);
  const label = methodKindLabel[method.kind];
  if (sameKind.length < 2) {
    return label;
  }
  return `${label} ${sameKind.findIndex((item) => item.id === method.id) + 1}`;
}

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
  const [addKind, setAddKind] = useState<MFAMethod['kind']>();
  const [showKey, setShowKey] = useState(false);
  const methods = state?.methods ?? [];
  const totpCount = methods.filter((method) => method.kind === 'totp').length;
  const passkeyCount = methods.filter(
    (method) => method.kind === 'passkey',
  ).length;
  const needsStepUp = enrolled && !state?.canManage && !setup;

  function closeAdd() {
    setAddKind(undefined);
    setSetup(undefined);
    setShowKey(false);
    setCode('');
    setRecovery(false);
    setError('');
  }

  function acknowledgeCodes() {
    setCodes(undefined);
    // Drop cached shard tokens so the next call is minted from the session
    // that just changed factors. Leave the rest of the cache in place: wiping
    // it makes the settings shell swap this page for its loading screen.
    queryClient.removeQueries({
      queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
    });
    closeAdd();
  }

  async function startAuthenticator() {
    setBusy(true);
    setError('');
    try {
      setSetup(
        await mfaPost<{ secret: string; uri: string; qr: string }>(
          'totp-enroll/start',
        ),
      );
    } catch {
      setError('Could not start authenticator setup.');
    } finally {
      setBusy(false);
    }
  }

  function choose(kind: MFAMethod['kind']) {
    setAddKind(kind);
    setCode('');
    setRecovery(false);
    setSetup(undefined);
    setError('');
    if (kind === 'totp' && !(enrolled && !state?.canManage)) {
      void startAuthenticator();
    }
  }

  const verificationExpired = error === 'invalid or expired MFA verification';

  async function retryVerification() {
    setCode('');
    setError('');
    if (needsStepUp) {
      return;
    }
    if (addKind === 'totp') {
      await startAuthenticator();
      return;
    }
    if (addKind === 'passkey') {
      const ok = await run(() => passkeyCeremony(true));
      if (ok) {
        closeAdd();
      }
    }
  }

  async function confirmStepUp(event: FormEvent) {
    event.preventDefault();
    const ok = await run(() =>
      mfaPost(`${recovery ? 'recovery' : 'totp'}/finish`, {
        code: code.trim(),
      }),
    );
    if (ok && addKind === 'totp') {
      await startAuthenticator();
    }
  }

  return (
    <div className="ph-no-capture ph-no-record h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="MFA"
          description="Add authenticator apps and passkeys. Any enrolled method can finish a Cloud login, including SSO and email links."
        />

        {status.isError && (
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>
              {error || 'Sign in again to manage your MFA.'}
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
        {state && (
          <div className="divide-y divide-border">
            <div className="py-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h2 className="text-sm font-medium">MFA methods</h2>
                  <p className="max-w-xl text-sm text-muted-foreground">
                    Add as many authenticator apps and passkeys as you use.
                    Passkeys do not replace your password or SSO login.
                  </p>
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      disabled={
                        busy ||
                        (totpCount >= methodLimit &&
                          passkeyCount >= methodLimit)
                      }
                    >
                      Add method
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      disabled={totpCount >= methodLimit}
                      onClick={() => choose('totp')}
                    >
                      Authenticator app
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={passkeyCount >= methodLimit}
                      onClick={() => choose('passkey')}
                    >
                      Passkey
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Method</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead className="text-right">
                      <span className="sr-only">Remove</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {methods.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} className="text-muted-foreground">
                        No MFA methods yet.
                      </TableCell>
                    </TableRow>
                  )}
                  {methods.map((method) => (
                    <TableRow key={`${method.kind}:${method.id}`}>
                      <TableCell>{methodName(method, methods)}</TableCell>
                      <TableCell>{methodTypeLabel[method.kind]}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={busy}
                          onClick={() => setRemoveFactor(method.id)}
                        >
                          Remove
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            {enrolled && (
              <SettingRow
                label="Recovery codes"
                description={`${state.recoveryRemaining} codes left. Each code works once, after your normal login.`}
              >
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => run(() => mfaPost('recovery'))}
                >
                  Replace recovery codes
                </Button>
              </SettingRow>
            )}
          </div>
        )}
      </div>

      <Dialog
        open={addKind !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            closeAdd();
          }
        }}
      >
        <DialogContent className="ph-no-capture ph-no-record w-fit min-w-[500px] max-w-[80%]">
          <DialogHeader className="text-left">
            <DialogTitle>
              {needsStepUp
                ? 'Confirm it\u2019s you'
                : addKind === 'passkey'
                  ? 'Add a passkey'
                  : 'Add an authenticator app'}
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-foreground">
            {needsStepUp
              ? 'Enter a code from a method you already have. Then you can add another.'
              : addKind === 'passkey'
                ? 'Your device will ask you to create a passkey. It does not replace your password or SSO login.'
                : 'Works with Google Authenticator, 1Password, Authy, and any app that supports time-based codes.'}
          </p>
          {error && (
            <Alert variant="destructive">
              <AlertDescription className="flex items-center justify-between gap-3">
                <span>{error}</span>
                {verificationExpired && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={() => void retryVerification()}
                  >
                    Retry
                  </Button>
                )}
              </AlertDescription>
            </Alert>
          )}
          {needsStepUp && (
            <form className="grid gap-3" onSubmit={confirmStepUp}>
              {state && state.passkeys.length > 0 && !recovery && (
                <Button
                  type="button"
                  disabled={busy}
                  onClick={async () => {
                    const ok = await run(() => passkeyCeremony(false));
                    if (ok && addKind === 'totp') {
                      await startAuthenticator();
                    }
                  }}
                >
                  Use a passkey
                </Button>
              )}
              {(state?.totp || recovery) && (
                <>
                  <Label htmlFor="mfa-step-up">
                    {recovery ? 'Recovery code' : 'Authenticator code'}
                  </Label>
                  <Input
                    id="mfa-step-up"
                    value={code}
                    onChange={(event) =>
                      setCode(
                        recovery
                          ? event.target.value
                          : event.target.value.replace(/\D/g, '').slice(0, 6),
                      )
                    }
                    autoComplete="one-time-code"
                    autoFocus
                    inputMode={recovery ? 'text' : 'numeric'}
                    maxLength={recovery ? 64 : 6}
                    placeholder={recovery ? undefined : '123456'}
                    disabled={busy}
                    className={
                      recovery
                        ? undefined
                        : 'w-40 font-mono text-lg tracking-[0.4em]'
                    }
                  />
                </>
              )}
              <div className="flex flex-row justify-end gap-4">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setRecovery(!recovery);
                    setCode('');
                  }}
                >
                  {recovery ? 'Use an enrolled method' : 'Use a recovery code'}
                </Button>
                <Button type="submit" disabled={busy || !code.trim()}>
                  Continue
                </Button>
              </div>
            </form>
          )}
          {!needsStepUp && addKind === 'passkey' && (
            <div className="flex flex-row justify-end gap-4">
              <Button type="button" variant="ghost" onClick={closeAdd}>
                Cancel
              </Button>
              <Button
                disabled={busy}
                onClick={async () => {
                  const ok = await run(() => passkeyCeremony(true));
                  if (ok) {
                    closeAdd();
                  }
                }}
              >
                Create passkey
              </Button>
            </div>
          )}
          {!needsStepUp && addKind === 'totp' && !setup && !error && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Spinner />
              Generating your setup key…
            </div>
          )}
          {!needsStepUp && addKind === 'totp' && setup && (
            <form
              className="grid gap-5"
              onSubmit={async (event) => {
                event.preventDefault();
                const ok = await run(() =>
                  mfaPost('totp-enroll/finish', { code: code.trim() }),
                );
                if (ok) {
                  closeAdd();
                }
              }}
            >
              <div className="grid gap-2">
                <p className="text-sm font-medium">
                  1. Scan this code with your authenticator app
                </p>
                <div className="flex flex-col items-center gap-3 sm:flex-row sm:items-start">
                  <img
                    src={setup.qr}
                    width={176}
                    height={176}
                    alt="Authenticator setup QR code"
                    className="rounded-md bg-white p-2"
                  />
                  <div className="grid flex-1 gap-2 text-sm">
                    <button
                      type="button"
                      className="w-fit text-left text-muted-foreground underline-offset-4 hover:underline"
                      onClick={() => setShowKey(!showKey)}
                    >
                      {showKey
                        ? 'Hide setup key'
                        : "Can't scan? Enter the key instead"}
                    </button>
                    {showKey && (
                      <div className="grid gap-1">
                        <div className="flex items-center gap-1 rounded-md border bg-muted/40 px-2 py-1.5">
                          <code className="flex-1 break-all font-mono text-xs tracking-wider">
                            {groupSecret(setup.secret)}
                          </code>
                          <CopyToClipboard text={setup.secret} />
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Time-based, 6 digits, 30 seconds.
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="setup-code" className="text-sm font-medium">
                  2. Enter the 6-digit code the app shows
                </Label>
                <Input
                  id="setup-code"
                  value={code}
                  onChange={(event) =>
                    setCode(event.target.value.replace(/\D/g, '').slice(0, 6))
                  }
                  inputMode="numeric"
                  pattern="[0-9]*"
                  autoComplete="one-time-code"
                  autoFocus
                  maxLength={6}
                  placeholder="123456"
                  disabled={busy}
                  className="w-40 font-mono text-lg tracking-[0.4em]"
                />
              </div>
              <div className="flex flex-row justify-end gap-4">
                <Button type="button" variant="ghost" onClick={closeAdd}>
                  Cancel
                </Button>
                <Button disabled={busy || code.length !== 6}>
                  {busy && <Spinner />}
                  Add authenticator
                </Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={!!codes}>
        <DialogContent
          className="ph-no-capture ph-no-record w-fit min-w-[500px] max-w-[80%]"
          onEscapeKeyDown={(event) => event.preventDefault()}
          onPointerDownOutside={(event) => event.preventDefault()}
          onInteractOutside={(event) => event.preventDefault()}
        >
          <DialogHeader className="text-left">
            <DialogTitle>Save your recovery codes</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-foreground">
            Keep these somewhere safe outside Hatchet. Each code works once,
            after your normal login. They will not be shown again.
          </p>
          {codes && (
            <pre className="overflow-auto text-xs select-all">
              {codes.join('\n')}
            </pre>
          )}
          <div className="flex flex-row justify-end gap-4">
            <Button
              variant="outline"
              onClick={() => {
                if (!codes) {
                  return;
                }
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
            <Button onClick={acknowledgeCodes}>I have saved these codes</Button>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        isOpen={removeFactor !== undefined}
        title="Remove this method?"
        description="Removing your last method turns MFA off. Other sessions will need to sign in again."
        submitLabel="Remove"
        cancelLabel="Keep it"
        isLoading={busy}
        onCancel={() => setRemoveFactor(undefined)}
        onSubmit={() => {
          void run(() => mfaPost('remove', { factor: removeFactor })).then(
            (ok) => {
              if (ok) {
                setRemoveFactor(undefined);
              }
            },
          );
        }}
      />
    </div>
  );
}
