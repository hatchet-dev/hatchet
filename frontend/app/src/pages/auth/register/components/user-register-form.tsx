import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  name: z.string().min(3, 'Name must be at least 3 characters long'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters long'),
});

type SubmitType = z.infer<typeof schema>;

interface UserRegisterFormProps {
  className?: string;
  onSubmit: (opts: SubmitType) => void;
  isLoading: boolean;
  errors?: string[];
  fieldErrors?: Record<string, string>;
}

export function UserRegisterForm({
  className,
  ...props
}: UserRegisterFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors, isValid, touchedFields, submitCount },
  } = useForm<SubmitType>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    reValidateMode: 'onChange',
  });

  const nameError =
    (touchedFields.name || submitCount > 0
      ? errors.name?.message?.toString()
      : undefined) || props.fieldErrors?.name;

  const emailError =
    (touchedFields.email || submitCount > 0
      ? errors.email?.message?.toString()
      : undefined) || props.fieldErrors?.email;

  const passwordError =
    (touchedFields.password || submitCount > 0
      ? errors.password?.message?.toString()
      : undefined) || props.fieldErrors?.password;

  return (
    <div className={cn('grid gap-6', className)}>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit(d);
        })}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input
              {...register('name')}
              id="name"
              placeholder="Boba Fett"
              type="name"
              autoCapitalize="none"
              autoCorrect="off"
              disabled={props.isLoading}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="email">Email</Label>
            <Input
              {...register('email')}
              id="email"
              placeholder="name@example.com"
              type="email"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              disabled={props.isLoading}
            />
            {emailError && (
              <div className="text-sm text-red-500">{emailError}</div>
            )}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="password">Password</Label>
            <Input
              {...register('password')}
              id="password"
              placeholder="Password"
              type="password"
              disabled={props.isLoading}
            />
            {passwordError && (
              <div className="text-sm text-red-500">{passwordError}</div>
            )}
          </div>
          {props.errors && props.errors.length > 0 && (
            <Alert variant="destructive">
              <AlertDescription>{props.errors.join(' ')}</AlertDescription>
            </Alert>
          )}
          <Button disabled={props.isLoading || !isValid}>
            {props.isLoading && <Spinner />}
            Create Account
          </Button>
        </div>
      </form>
    </div>
  );
}
