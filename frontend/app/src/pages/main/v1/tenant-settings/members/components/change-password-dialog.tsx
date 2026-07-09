import { useToast } from '@/components/v1/hooks/use-toast';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z
  .object({
    password: z.string().min(1, 'Enter your current password'),
    newPassword: z
      .string()
      .min(8, 'New password must be at least 8 characters'),
    confirmNewPassword: z.string(),
  })
  .refine((data) => data.newPassword === data.confirmNewPassword, {
    message: 'Passwords do not match',
    path: ['confirmNewPassword'],
  });

export function ChangePasswordDialog({ onClose }: { onClose: () => void }) {
  const { toast } = useToast();
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });
  const { userUpdatePasswordMutation } = useUserApi();

  const updateMutation = useMutation({
    ...userUpdatePasswordMutation(),
    onSuccess: () => {
      toast({
        title: 'Password updated',
        duration: 4000,
      });
      onClose();
    },
    onError: handleApiError,
  });

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  const passwordError = errors.password?.message?.toString();
  const newPasswordError = errors.newPassword?.message?.toString();
  const confirmError = errors.confirmNewPassword?.message?.toString();

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Change password</DialogTitle>
          <DialogDescription>
            Update the password you use to sign in.
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={handleSubmit((data) => {
            setFormErrors([]);
            updateMutation.mutate({
              password: data.password,
              newPassword: data.newPassword,
            });
          })}
        >
          <div className="grid gap-4">
            <InlineError errors={formErrors} />
            <div className="grid gap-2">
              <Label htmlFor="current-password">Current password</Label>
              <Input
                {...register('password')}
                id="current-password"
                type="password"
                autoComplete="current-password"
                disabled={updateMutation.isPending}
              />
              {passwordError && (
                <div className="text-sm text-red-500">{passwordError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="new-password">New password</Label>
              <Input
                {...register('newPassword')}
                id="new-password"
                type="password"
                autoComplete="new-password"
                disabled={updateMutation.isPending}
              />
              {newPasswordError && (
                <div className="text-sm text-red-500">{newPasswordError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="confirm-new-password">Confirm new password</Label>
              <Input
                {...register('confirmNewPassword')}
                id="confirm-new-password"
                type="password"
                autoComplete="new-password"
                disabled={updateMutation.isPending}
              />
              {confirmError && (
                <div className="text-sm text-red-500">{confirmError}</div>
              )}
            </div>
            <Button disabled={updateMutation.isPending}>
              {updateMutation.isPending && <Spinner />}
              Update password
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
