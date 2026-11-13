import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email('Invalid email address'),
});

export function MagicLinkLogin() {
  const [errors, setErrors] = useState<string[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [sentTo, setSentTo] = useState<string | null>(null);
  const [resendAt, setResendAt] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const { handleApiError } = useApiError({ setFieldErrors, setErrors });
  const { userCreateMagicLinkMutation } = useUserApi();

  const resendSecondsLeft = Math.max(0, Math.ceil((resendAt - now) / 1000));
  const coolingDown = resendSecondsLeft > 0;

  useEffect(() => {
    if (!coolingDown) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [coolingDown]);
  const {
    register,
    handleSubmit,
    formState: { errors: formErrors, isValid, touchedFields, submitCount },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    reValidateMode: 'onChange',
  });

  const requestLink = useMutation({
    ...userCreateMagicLinkMutation(),
    onSuccess: (data, variables) => {
      const sentAt = Date.now();
      setSentTo(variables.email);
      setNow(sentAt);
      setResendAt(sentAt + data.resendCooldownSeconds * 1000);
    },
    onError: handleApiError,
  });

  const emailError =
    (touchedFields.email || submitCount > 0
      ? formErrors.email?.message?.toString()
      : undefined) || fieldErrors.email;

  if (sentTo) {
    return (
      <div className="grid gap-4 text-sm">
        <p>
          If this address can sign in with a login link, we sent one to{' '}
          <span className="font-medium">{sentTo}</span>. It expires soon and can
          only be used once.
        </p>
        <Button
          type="button"
          variant="outline"
          disabled={requestLink.isPending || coolingDown}
          onClick={() => {
            setErrors([]);
            requestLink.mutate({ email: sentTo });
          }}
        >
          {requestLink.isPending && <Spinner />}
          {coolingDown
            ? `Send another link in ${resendSecondsLeft}s`
            : 'Send another link'}
        </Button>
        {errors.length > 0 && (
          <Alert variant="destructive">
            <AlertDescription>{errors.join(' ')}</AlertDescription>
          </Alert>
        )}
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit((data) => {
        setErrors([]);
        setFieldErrors({});
        requestLink.mutate(data);
      })}
    >
      <div className="grid gap-4">
        <div className="grid gap-2">
          <Label htmlFor="magic-link-email">Email</Label>
          <Input
            {...register('email')}
            id="magic-link-email"
            autoFocus
            placeholder="name@example.com"
            type="email"
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect="off"
            disabled={requestLink.isPending}
          />
          <p className="text-sm text-muted-foreground">
            We&apos;ll email a one-time link. If you don&apos;t have an account
            yet, this creates one.
          </p>
          {emailError && (
            <div className="text-sm text-red-500">{emailError}</div>
          )}
        </div>
        {errors.length > 0 && (
          <Alert variant="destructive">
            <AlertDescription>{errors.join(' ')}</AlertDescription>
          </Alert>
        )}
        <Button disabled={requestLink.isPending || !isValid}>
          {requestLink.isPending && <Spinner />}
          Email me a login link
        </Button>
      </div>
    </form>
  );
}
