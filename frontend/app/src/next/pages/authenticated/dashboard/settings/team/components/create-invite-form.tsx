import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { Label } from '@/next/components/ui/label';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { TenantMemberRole } from '@/next/lib/api/generated/data-contracts';
import { useState, useEffect, useMemo, useCallback } from 'react';
import useMembers from '@/next/hooks/use-members';
import { SendIcon } from 'lucide-react';
import useCan from '@/next/hooks/use-can';
import { members } from '@/next/lib/can/features/members.permissions';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';

const schema = z.object({
  emails: z
    .string()
    .min(1, 'At least one email address is required')
    .refine(
      (emails) => {
        const emailList = emails
          .split(',')
          .map((email) => email.trim())
          .filter((email) => email.length > 0);
        return emailList.every((email) =>
          /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email),
        );
      },
      { message: 'One or more email addresses are invalid' },
    ),
  role: z.enum([
    TenantMemberRole.OWNER,
    TenantMemberRole.ADMIN,
    TenantMemberRole.MEMBER,
  ]),
});

interface CreateInviteFormProps {
  className?: string;
  close: () => void;
}

// All available roles in descending order of privilege
const ALL_ROLES = [
  TenantMemberRole.OWNER,
  TenantMemberRole.ADMIN,
  TenantMemberRole.MEMBER,
];

export function CreateInviteForm({ className, close }: CreateInviteFormProps) {
  const { invite } = useMembers();
  const { canWithReason } = useCan();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // Function to check if a role is allowed
  const isRoleAllowed = useCallback(
    (role: TenantMemberRole) => {
      return canWithReason(members.invite(role)).allowed;
    },
    [canWithReason],
  );

  // Get the list of allowed roles
  const allowedRoles = useMemo(() => {
    return ALL_ROLES.filter(isRoleAllowed);
  }, [isRoleAllowed]);

  // Find the highest role the user can invite
  const getDefaultRole = useCallback(() => {
    return allowedRoles[0] || TenantMemberRole.MEMBER; // Default to MEMBER if nothing is allowed
  }, [allowedRoles]);

  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      emails: '',
      role: getDefaultRole(),
    },
  });

  // If the currently selected role becomes unavailable, update to highest available
  const selectedRole = watch('role');
  useEffect(() => {
    if (!isRoleAllowed(selectedRole)) {
      setValue('role', getDefaultRole());
    }
  }, [selectedRole, setValue, allowedRoles, getDefaultRole, isRoleAllowed]);

  // Watch emails to update button text
  const emails = watch('emails');
  const emailCount = emails
    ? emails
        .split(',')
        .map((email) => email.trim())
        .filter((email) => email.length > 0).length
    : 0;

  const emailsError = errors.emails?.message?.toString() || fieldErrors.emails;
  const roleError = errors.role?.message?.toString() || fieldErrors.role;

  const onSubmit = async (data: z.infer<typeof schema>) => {
    const emailList = data.emails
      .split(',')
      .map((email) => email.trim())
      .filter((email) => email.length > 0);

    // Create an array of promises for each invite
    const invitePromises = emailList.map(
      (email) =>
        new Promise((resolve, reject) => {
          invite.mutate(
            { email, role: data.role },
            {
              onSuccess: () => resolve(email),
              onError: (error) => reject({ email, error }),
            },
          );
        }),
    );

    try {
      await Promise.all(invitePromises);
      close();
      setFieldErrors({});
    } catch (error: any) {
      // Handle errors from any of the invites
      if (error?.error?.response?.data?.errors) {
        const apiErrors = error.error.response.data.errors;
        const newFieldErrors: Record<string, string> = {};
        apiErrors.forEach((err: any) => {
          if (err.field) {
            newFieldErrors[err.field] = `${error.email}: ${err.message}`;
          }
        });
        setFieldErrors(newFieldErrors);
      }
    }
  };

  // Get message why a role is disabled
  const getRoleDisabledMessage = (role: TenantMemberRole) => {
    const { message } = canWithReason(members.invite(role));
    return (
      message || 'You do not have permission to invite users with this role'
    );
  };

  // Format the role for display
  const formatRole = (role: TenantMemberRole) => {
    switch (role) {
      case TenantMemberRole.OWNER:
        return 'Owner';
      case TenantMemberRole.ADMIN:
        return 'Admin';
      case TenantMemberRole.MEMBER:
        return 'Member';
      default:
        return role;
    }
  };

  return (
    <DialogContent className="max-w-md">
      <DialogHeader>
        <DialogTitle>Invite New Member{emailCount > 1 ? 's' : ''}</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="emails">Email{emailCount > 1 ? 's' : ''}</Label>
              <textarea
                {...register('emails')}
                id="emails"
                placeholder="name@example.com, name2@example.com"
                rows={3}
                className="resize-none flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={invite.isPending}
              />
              {emailsError && (
                <div className="text-sm text-destructive">{emailsError}</div>
              )}
              <p className="text-xs text-muted-foreground">
                Enter one or more email addresses separated by commas
              </p>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="role">Role</Label>
              <Controller
                control={control}
                name="role"
                render={({ field }) => (
                  <Select onValueChange={field.onChange} {...field}>
                    <SelectTrigger>
                      <SelectValue id="role" placeholder="Role..." />
                    </SelectTrigger>
                    <SelectContent>
                      <TooltipProvider>
                        {ALL_ROLES.map((role) => {
                          const isAllowed = isRoleAllowed(role);

                          if (!isAllowed) {
                            return (
                              <Tooltip key={role}>
                                <TooltipTrigger asChild>
                                  <div>
                                    <SelectItem
                                      value={role}
                                      disabled={true}
                                      className="cursor-not-allowed opacity-50"
                                    >
                                      {formatRole(role)}
                                    </SelectItem>
                                  </div>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <p>{getRoleDisabledMessage(role)}</p>
                                </TooltipContent>
                              </Tooltip>
                            );
                          }

                          return (
                            <SelectItem key={role} value={role}>
                              {formatRole(role)}
                            </SelectItem>
                          );
                        })}
                      </TooltipProvider>
                    </SelectContent>
                  </Select>
                )}
              />
              {roleError && (
                <div className="text-sm text-destructive">{roleError}</div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={close}
                disabled={invite.isPending}
              >
                Cancel
              </Button>
              <Button loading={invite.isPending}>
                <SendIcon className="mr-2 h-4 w-4" />
                Send Invite{emailCount > 1 ? 's' : ''}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
