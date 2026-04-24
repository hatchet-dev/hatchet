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
import {
  CreateOrganizationInviteRequest,
  OrganizationMemberRoleType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { UserPlusIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email('Invalid email address'),
});

type OrganizationInviteMemberModalProps = {
  organizationId: string;
  organizationName: string;
  onClose: () => void;
  onCreated: (invite: CreateOrganizationInviteRequest) => void;
};

export const OrganizationInviteMemberModal = ({
  organizationId,
  organizationName,
  onClose,
  onCreated,
}: OrganizationInviteMemberModalProps) => {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const { handleApiError } = useApiError({
    setFieldErrors,
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

  const queryClient = useQueryClient();
  const orgApi = useOrganizationApi();
  const orgInviteCreate =
    orgApi.organizationInviteCreateMutation(organizationId);
  const inviteMemberMutation = useMutation({
    ...orgInviteCreate,
    mutationFn: async (data: { email: string }) => {
      const request: CreateOrganizationInviteRequest = {
        inviteeEmail: data.email,
        role: OrganizationMemberRoleType.OWNER,
      };
      await orgInviteCreate.mutationFn(request);
      return request;
    },
    onSuccess: (request) => {
      queryClient.invalidateQueries({
        queryKey: ['organization-invites:list', organizationId],
      });
      reset();
      onCreated(request);
      onClose();
    },
    onError: handleApiError,
  });

  const emailError = errors.email?.message?.toString() || fieldErrors?.email;

  useEffect(() => {
    reset();
    setFieldErrors({});
  }, [reset]);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
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
            <Button type="button" variant="outline" onClick={onClose}>
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
};
