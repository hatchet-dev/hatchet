import { SettingsPageHeader } from '../../tenant-settings/components/settings-page-header';
import { SettingRow } from '../../tenant-settings/components/settings-row';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
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
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableRow,
} from '@/components/v1/ui/table';
import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import { type MFAMethod } from '@/lib/api/mfa';
import { EXPIRED_MESSAGE, leaveMfa, useMfa } from '@/pages/auth/mfa/use-mfa';
import queryClient from '@/query-client';
import { type FormEvent, useState } from 'react';

const methodLimit = 10;

const methodKindLabel: Record<MFAMethod['kind'], string> = {
  totp: 'Authenticator app',
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

function StepUpForm({
  id,
  hasPasskey,
  hasTotp,
  recovery,
  setRecovery,
  code,
  setCode,
  busy,
  onPasskey,
  onSubmit,
}: {
  id: string;
  hasPasskey: boolean;
  hasTotp: boolean;
  recovery: boolean;
  setRecovery: (recovery: boolean) => void;
  code: string;
  setCode: (code: string) => void;
  busy: boolean;
  onPasskey: () => void;
  onSubmit: (event: FormEvent) => void;
}) {
  return (
    <form className="grid gap-3" onSubmit={onSubmit}>
      {hasPasskey && !recovery && (
        <Button type="button" disabled={busy} onClick={onPasskey}>
          Use a passkey
        </Button>
      )}
      {(hasTotp || recovery) && (
        <>
          <Label htmlFor={id}>
            {recovery ? 'Recovery code' : 'Authenticator code'}
          </Label>
          <Input
            id={id}
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
              recovery ? undefined : 'w-40 font-mono text-lg tracking-[0.4em]'
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
          {recovery ? 'Use a passkey or app' : 'Use a recovery code'}
        </Button>
        <Button type="submit" disabled={busy || !code.trim()}>
          Continue
        </Button>
      </div>
    </form>
  );
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
  // A fresh dialog asks for an enrolled factor even when this session proved one recently.
  const [stepUpDone, setStepUpDone] = useState(false);
  const [removeProved, setRemoveProved] = useState(false);
  const [replacingCodes, setReplacingCodes] = useState(false);
  const [replaceProved, setReplaceProved] = useState(false);
  const methods = state?.methods ?? [];
  const removing = methods.find((method) => method.id === removeFactor);
  const removingLast = methods.length === 1 && removing !== undefined;
  const totpCount = methods.filter((method) => method.kind === 'totp').length;
  const passkeyCount = methods.filter(
    (method) => method.kind === 'passkey',
  ).length;
  const needsStepUp = enrolled && !stepUpDone && !setup;
  const removeNeedsChallenge =
    removeFactor !== undefined && enrolled && !removeProved;
  const replaceNeedsChallenge = replacingCodes && enrolled && !replaceProved;
  const stepUpProps = {
    hasPasskey: (state?.passkeys.length ?? 0) > 0,
    hasTotp: state?.totp ?? false,
    recovery,
    setRecovery,
    code,
    setCode,
    busy,
  };

  function closeAdd() {
    setAddKind(undefined);
    setSetup(undefined);
    setShowKey(false);
    setStepUpDone(false);
    setCode('');
    setRecovery(false);
    setError('');
  }

  function closeRemove() {
    setRemoveFactor(undefined);
    setRemoveProved(false);
    setCode('');
    setRecovery(false);
    setError('');
  }

  function closeReplace() {
    setReplacingCodes(false);
    setReplaceProved(false);
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
      setError("Couldn't start setup. Try again.");
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
    if (kind === 'totp' && !enrolled) {
      void startAuthenticator();
    }
  }

  async function proveFactor() {
    return run(() =>
      mfaPost(`${recovery ? 'recovery' : 'totp'}/finish`, {
        code: code.trim(),
      }),
    );
  }

  const verificationExpired = error === EXPIRED_MESSAGE;

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
    const ok = await proveFactor();
    if (!ok) {
      return;
    }
    setStepUpDone(true);
    if (addKind === 'totp') {
      await startAuthenticator();
    }
  }

  async function confirmRemoveStepUp(event: FormEvent) {
    event.preventDefault();
    const ok = await proveFactor();
    if (ok) {
      setRemoveProved(true);
    }
  }

  async function confirmReplaceStepUp(event: FormEvent) {
    event.preventDefault();
    const ok = await proveFactor();
    if (ok) {
      setReplaceProved(true);
    }
  }

  return (
    <div className="ph-no-capture ph-no-record h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Multifactor Auth"
          description="Add a passkey or authenticator app to verify it's you after you log in."
        />

        {status.isError && (
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>
              {error || 'Log in again to manage multifactor auth.'}
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
            Log in again
          </Button>
        )}
        {state && (
          <div className="divide-y divide-border">
            <div className="py-4">
              <div className="flex items-center justify-between gap-3 border-b pb-3">
                <h2 className="text-sm font-medium">Methods</h2>
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
                      disabled={passkeyCount >= methodLimit}
                      onClick={() => choose('passkey')}
                    >
                      Passkey
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={totpCount >= methodLimit}
                      onClick={() => choose('totp')}
                    >
                      Authenticator app
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <Table>
                <TableBody>
                  {methods.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={2} className="py-8">
                        <EmptyState
                          title="No methods added"
                          description="Add a passkey or authenticator app to turn on multifactor auth."
                        />
                      </TableCell>
                    </TableRow>
                  )}
                  {methods.map((method) => (
                    <TableRow key={`${method.kind}:${method.id}`}>
                      <TableCell>{methodName(method, methods)}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={busy}
                          onClick={() => {
                            setCode('');
                            setRecovery(false);
                            setError('');
                            setRemoveProved(false);
                            setRemoveFactor(method.id);
                          }}
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
                label="Recovery Codes"
                description={`${state.recoveryRemaining} unused. If you lose your other methods, each code works for one login.`}
              >
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => setReplacingCodes(true)}
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
              ? 'Verify with a method you already use before adding another.'
              : addKind === 'passkey'
                ? 'Your browser will ask you to create a passkey on this device or a security key.'
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
            <StepUpForm
              id="mfa-step-up"
              {...stepUpProps}
              onSubmit={confirmStepUp}
              onPasskey={async () => {
                const ok = await run(() => passkeyCeremony(false));
                if (!ok) {
                  return;
                }
                setStepUpDone(true);
                if (addKind === 'totp') {
                  await startAuthenticator();
                }
              }}
            />
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
            Keep these somewhere safe outside Hatchet. Use them once each if you
            lose your other methods. You won&apos;t see them again.
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

      <Dialog
        open={removeFactor !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            closeRemove();
          }
        }}
      >
        <DialogContent className="ph-no-capture ph-no-record w-fit min-w-[500px] max-w-[80%]">
          <DialogHeader className="text-left">
            <DialogTitle>
              {removeNeedsChallenge
                ? 'Confirm it\u2019s you'
                : `Remove ${removing ? methodName(removing, methods) : 'this method'}?`}
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-foreground">
            {removeNeedsChallenge
              ? 'Verify with a method you already use before removing one.'
              : removingLast
                ? 'This turns off multifactor auth and logs out your other sessions.'
                : 'Your other sessions will be logged out.'}
          </p>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {removeNeedsChallenge && (
            <StepUpForm
              id="mfa-remove-step-up"
              {...stepUpProps}
              onSubmit={confirmRemoveStepUp}
              onPasskey={async () => {
                const ok = await run(() => passkeyCeremony(false));
                if (ok) {
                  setRemoveProved(true);
                }
              }}
            />
          )}
          {!removeNeedsChallenge && (
            <div className="flex flex-row justify-end gap-4">
              <Button type="button" variant="ghost" onClick={closeRemove}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                disabled={busy}
                onClick={() => {
                  void run(() =>
                    mfaPost('remove', { factor: removeFactor }),
                  ).then((ok) => {
                    if (ok) {
                      closeRemove();
                    }
                  });
                }}
              >
                {busy && <Spinner />}
                Remove
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={replacingCodes}
        onOpenChange={(open) => {
          if (!open) {
            closeReplace();
          }
        }}
      >
        <DialogContent className="ph-no-capture ph-no-record w-fit min-w-[500px] max-w-[80%]">
          <DialogHeader className="text-left">
            <DialogTitle>
              {replaceNeedsChallenge
                ? 'Confirm it\u2019s you'
                : 'Replace recovery codes?'}
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-foreground">
            {replaceNeedsChallenge
              ? 'Verify with a method you already use before replacing your codes.'
              : 'Your current codes will stop working.'}
          </p>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {replaceNeedsChallenge && (
            <StepUpForm
              id="mfa-replace-step-up"
              {...stepUpProps}
              onSubmit={confirmReplaceStepUp}
              onPasskey={async () => {
                const ok = await run(() => passkeyCeremony(false));
                if (ok) {
                  setReplaceProved(true);
                }
              }}
            />
          )}
          {!replaceNeedsChallenge && (
            <div className="flex flex-row justify-end gap-4">
              <Button type="button" variant="ghost" onClick={closeReplace}>
                Cancel
              </Button>
              <Button
                disabled={busy}
                onClick={() => {
                  void run(() => mfaPost('recovery')).then((ok) => {
                    if (ok) {
                      closeReplace();
                    }
                  });
                }}
              >
                {busy && <Spinner />}
                Replace codes
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
