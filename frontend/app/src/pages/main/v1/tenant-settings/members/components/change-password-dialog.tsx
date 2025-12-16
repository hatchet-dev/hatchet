import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Button } from '@/components/v1/ui/button';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/v1/ui/label';
import { Input } from '@/components/v1/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/v1/ui/loading';

const passwordSchema = z
  .string()
  .min(8, 'Passwords must be at least 8 characters in length')
  .max(255)
  .regex(
    /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/,
    'Passwords must contain an upper and lowercase letter, and at least one number',
  );

const schema = z
  .object({
    password: z.string(),
    newPassword: passwordSchema,
    confirmNewPassword: z.string(),
  })
  .superRefine(({ newPassword, confirmNewPassword }, ctx) => {
    if (newPassword !== confirmNewPassword) {
      ctx.addIssue({
        code: 'custom',
        message: 'Passwords do not match',
      });
    }
  });

interface ChangePasswordDialogProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function ChangePasswordDialog({
  className,
  ...props
}: ChangePasswordDialogProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  // @ts-expect-error - zod doesn't have a good way to get the global error
  const globalError = errors[''] as z.ZodIssue;

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Change Password</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="email">Current Password</Label>
              <Input
                {...register('password')}
                id="current-password"
                autoCapitalize="none"
                autoCorrect="off"
                type="password"
                disabled={props.isLoading}
              />
              {errors.password && (
                <div className="text-sm text-red-500">
                  {errors.password.message}
                </div>
              )}
              {props.fieldErrors?.password && (
                <div className="text-sm text-red-500">
                  {props.fieldErrors?.password}
                </div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">New Password</Label>
              <Input
                {...register('newPassword')}
                id="new-password"
                autoCapitalize="none"
                autoCorrect="off"
                type="password"
                disabled={props.isLoading}
              />
              {errors.newPassword && (
                <div className="text-sm text-red-500">
                  {errors.newPassword.message}
                </div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">Confirm New Password</Label>
              <Input
                {...register('confirmNewPassword')}
                id="confirm-new-password"
                autoCapitalize="none"
                autoCorrect="off"
                type="password"
                disabled={props.isLoading}
              />
              {errors.confirmNewPassword && (
                <div className="text-sm text-red-500">
                  {errors.confirmNewPassword.message}
                </div>
              )}
              {globalError && (
                <div className="text-sm text-red-500">
                  {globalError.message}
                </div>
              )}
            </div>
            <Button disabled={props.isLoading} type="submit">
              {props.isLoading && <Spinner />}
              Reset Password
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
