import { cn } from '@/next/lib/utils/index.ts';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import { Label } from '@/next/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';

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
  fieldErrors?: Record<string, string>;
  apiError?: string;
}

export function UserRegisterForm({
  className,
  ...props
}: UserRegisterFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SubmitType>({
    resolver: zodResolver(schema),
  });

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;

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
            {nameError ? (
              <div className="text-sm text-red-500">{nameError}</div>
            ) : null}
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
            {emailError ? (
              <div className="text-sm text-red-500">{emailError}</div>
            ) : null}
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
            {passwordError ? (
              <div className="text-sm text-red-500">{passwordError}</div>
            ) : null}
          </div>
          <Button loading={props.isLoading}>Create Account</Button>
          {props.apiError ? (
            <Alert variant="destructive">
              <AlertTitle>Registration Request Failed</AlertTitle>
              <AlertDescription>{props.apiError}</AlertDescription>
            </Alert>
          ) : null}
        </div>
      </form>
    </div>
  );
}
