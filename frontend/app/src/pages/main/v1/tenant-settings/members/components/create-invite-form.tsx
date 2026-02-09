import { Button } from '@/components/v1/ui/button';
import {
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
import { TenantMemberRole } from '@/lib/api';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

interface CreateInviteFormProps {
  className?: string;
  onSubmit: (opts: { email: string; role: TenantMemberRole }) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  isCloudEnabled?: boolean;
  organizationId?: string | null;
}

export function CreateInviteForm({
  className,
  ...props
}: CreateInviteFormProps) {
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
        <DialogTitle>Invite new member</DialogTitle>
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
              {props.isCloudEnabled && props.organizationId && (
                <div className="text-sm text-muted-foreground">
                  Organization owner invitations have moved to{' '}
                  <a
                    href={`/organizations/${props.organizationId}`}
                    className="text-primary hover:underline"
                  >
                    organization settings
                  </a>
                  .
                </div>
              )}
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
}
