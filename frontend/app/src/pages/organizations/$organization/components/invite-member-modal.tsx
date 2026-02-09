import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { cloudApi } from '@/lib/api/api';
import { OrganizationMemberRoleType } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { UserPlusIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email('Invalid email address'),
});

interface InviteMemberModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  organizationName: string;
  onSuccess: () => void;
}

export function InviteMemberModal({
  open,
  onOpenChange,
  organizationId,
  organizationName,
  onSuccess,
}: InviteMemberModalProps) {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: '',
    },
  });

  const inviteMemberMutation = useMutation({
    mutationFn: async (data: { email: string }) => {
      const result = await cloudApi.organizationInviteCreate(organizationId, {
        inviteeEmail: data.email,
        role: OrganizationMemberRoleType.OWNER,
      });
      return result.data;
    },
    onSuccess: () => {
      reset();
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const emailError = errors.email?.message?.toString() || fieldErrors?.email;

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      reset();
      setFieldErrors({});
    }
  }, [open, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlusIcon className="h-5 w-5" />
            Invite Member
          </DialogTitle>
          <DialogDescription>
            Invite a new member to {organizationName}
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit((data) => inviteMemberMutation.mutate(data))}
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label htmlFor="email">Email Address</Label>
            <Input
              {...register('email')}
              id="email"
              type="email"
              placeholder="name@example.com"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              disabled={inviteMemberMutation.isPending}
            />
            {emailError && (
              <div className="text-sm text-red-500">{emailError}</div>
            )}
            <p className="text-sm text-muted-foreground">
              The user will receive an email invitation to join this
              organization.
            </p>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={inviteMemberMutation.isPending}>
              {inviteMemberMutation.isPending
                ? 'Sending...'
                : 'Send Invitation'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
