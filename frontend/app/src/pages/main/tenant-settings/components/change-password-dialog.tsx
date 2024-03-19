import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/ui/loading';

const schema = z.object({
  password: z.string().min(1).max(255),
  newPassword: z.string().min(1).max(255),
  confirmNewPassword: z.string().min(1).max(255),
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
                disabled={props.isLoading}
              />
              {errors.password && (
                <div className="text-sm text-red-500">{errors.password.message}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">New Password</Label>
              <Input
                {...register('newPassword')}
                id="new-password"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {errors.newPassword && (
                <div className="text-sm text-red-500">{errors.newPassword.message}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">Confirm New Password</Label>
              <Input
                {...register('confirmNewPassword')}
                id="confirm-new-password"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {errors.confirmNewPassword && (
                <div className="text-sm text-red-500">{errors.confirmNewPassword.message}</div>
              )}
            </div>
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Reset Password
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
