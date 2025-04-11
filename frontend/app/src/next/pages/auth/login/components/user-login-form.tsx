import { cn } from '@/next/lib/utils/index.ts';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import { Label } from '@/next/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Alert, AlertTitle, AlertDescription } from '@/next/components/ui/alert';

const schema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters long'),
});

interface UserLoginFormProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  apiError?: string;
}

export function UserLoginForm({ className, ...props }: UserLoginFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  const emailError =
    errors.email?.message?.toString() || props.fieldErrors?.email;

  const passwordError =
    errors.password?.message?.toString() || props.fieldErrors?.password;

  return (
    <div className={cn('grid gap-6', className)}>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit(d);
        })}
      >
        <div className="grid gap-4">
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
          <Button disabled={props.isLoading}>Sign In</Button>
          {props.apiError && (
            <Alert variant="destructive">
              <AlertTitle>Login Request Failed</AlertTitle>
              <AlertDescription>
                <code>{props.apiError}</code>
              </AlertDescription>
            </Alert>
          )}
        </div>
      </form>
    </div>
  );
}
