import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
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
import useControlPlane from '@/hooks/use-control-plane';
import { TenantMember, TenantMemberRole } from '@/lib/api';
import { cn } from '@/lib/utils';
import { payloadsLockedForRole } from '@/pages/main/v1/tenant-settings/components/member-primitives';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  role: z.nativeEnum(TenantMemberRole),
  canViewPayloads: z.boolean(),
});

interface UpdateMemberFormProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  member: TenantMember;
  canSetOwnerRole?: boolean;
  formErrors?: string[];
}

export function UpdateMemberForm({
  className,
  ...props
}: UpdateMemberFormProps) {
  const { isControlPlaneEnabled } = useControlPlane();
  const {
    handleSubmit,
    control,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      role: props.member.role,
      canViewPayloads: props.member.canViewPayloads ?? true,
    },
  });

  // Keep the stored flag intact while OWNER/ADMIN are selected so switching
  // back to MEMBER/VIEWER restores the prior choice. Force true only in the
  // checkbox UI and on submit.
  const payloadsLocked = payloadsLockedForRole(watch('role'));

  const roleError = errors.role?.message?.toString();

  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Update member role</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit({
              ...d,
              canViewPayloads: payloadsLockedForRole(d.role)
                ? true
                : d.canViewPayloads,
            });
          })}
        >
          <div className="grid gap-4">
            <InlineError errors={props.formErrors ?? []} />
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                readOnly
                id="name"
                value={props.member.user.name || ''}
                disabled={true}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                readOnly
                id="email"
                placeholder="name@example.com"
                type="email"
                value={props.member.user.email}
                autoCapitalize="none"
                autoComplete="email"
                autoCorrect="off"
                disabled={true}
              />
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
                        {(!isControlPlaneEnabled || props.canSetOwnerRole) && (
                          <SelectItem value="OWNER">Owner</SelectItem>
                        )}
                        <SelectItem value="ADMIN">Admin</SelectItem>
                        <SelectItem value="MEMBER">Member</SelectItem>
                        <SelectItem value="VIEWER">Viewer</SelectItem>
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {roleError && (
                <div className="text-sm text-red-500">{roleError}</div>
              )}
            </div>
            <div className="flex items-start gap-2">
              <Controller
                control={control}
                name="canViewPayloads"
                render={({ field }) => (
                  <Checkbox
                    id="canViewPayloads"
                    checked={payloadsLocked || field.value}
                    onCheckedChange={(checked) =>
                      field.onChange(checked === true)
                    }
                    disabled={props.isLoading || payloadsLocked}
                  />
                )}
              />
              <div className="grid gap-1">
                <Label htmlFor="canViewPayloads">Can view payloads</Label>
                <p className="text-xs text-muted-foreground">
                  Owners and admins always see payloads. Uncheck to hide inputs,
                  outputs, and events from this member.
                </p>
              </div>
            </div>
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Update member
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
