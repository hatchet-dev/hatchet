import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useOrganizations } from '@/hooks/use-organizations';
import { TenantMemberRole } from '@/lib/api';
import { TenantInvite } from '@/lib/api/generated/data-contracts';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

type CreateTenantInviteFormProps = {
  className?: string;
  onSubmit: (opts: { email: string; role: TenantMemberRole }) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  isCloudEnabled?: boolean;
  organizationId?: string | null;
};

const CreateTenantInviteForm = ({
  className,
  ...props
}: CreateTenantInviteFormProps) => {
  const availableRoles = props.isCloudEnabled
    ? [TenantMemberRole.ADMIN, TenantMemberRole.MEMBER]
    : [TenantMemberRole.OWNER, TenantMemberRole.ADMIN, TenantMemberRole.MEMBER];

  const schema = z.object({
    email: z.string().email('Invalid email address'),
    role: z.enum(availableRoles as [TenantMemberRole, ...TenantMemberRole[]]),
  });

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  const emailError =
    errors.email?.message?.toString() || props.fieldErrors?.email;

  const roleError =
    errors.role?.message?.toString() || props.fieldErrors?.password;

  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Invite new tenant member</DialogTitle>
      </DialogHeader>
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
              <Label htmlFor="role">Role</Label>
              <Controller
                control={control}
                name="role"
                render={({ field }) => {
                  return (
                    <Select onValueChange={field.onChange} {...field}>
                      <SelectTrigger className="w-[180px]">
                        <SelectValue id="role" placeholder="Role..." />
                      </SelectTrigger>
                      <SelectContent>
                        {!props.isCloudEnabled && (
                          <SelectItem value="OWNER">Owner</SelectItem>
                        )}
                        <SelectItem value="ADMIN">Admin</SelectItem>
                        <SelectItem value="MEMBER">Member</SelectItem>
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {roleError && (
                <div className="text-sm text-red-500">{roleError}</div>
              )}
            </div>
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Invite user
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
};

export const CreateTenantInviteModal = ({
  tenantId,
  onClose,
  onCreated,
}: {
  tenantId: string;
  onClose: () => void;
  onCreated: (invite: TenantInvite) => void;
}) => {
  const { getOrganizationIdForTenant, isCloudEnabled } = useOrganizations();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors,
  });

  const queryClient = useQueryClient();
  const organizationId = getOrganizationIdForTenant(tenantId);

  const { tenantInviteCreateMutation } = useTenantApi();
  const createMutation = useMutation({
    ...tenantInviteCreateMutation(tenantId),
    onSuccess: (invite) => {
      queryClient.invalidateQueries({
        queryKey: ['tenant-invite:list', tenantId],
      });
      onCreated(invite);
      onClose();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <CreateTenantInviteForm
        isLoading={createMutation.isPending}
        onSubmit={createMutation.mutate}
        fieldErrors={fieldErrors}
        isCloudEnabled={isCloudEnabled}
        organizationId={organizationId}
      />
    </Dialog>
  );
};
