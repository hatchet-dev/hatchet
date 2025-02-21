import { cn } from '@/lib/utils';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { TenantMemberRole } from '@/lib/api';

const schema = z.object({
  email: z.string().email('Invalid email address'),
  role: z.enum([
    TenantMemberRole.OWNER,
    TenantMemberRole.ADMIN,
    TenantMemberRole.MEMBER,
  ]),
});

interface CreateInviteFormProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateInviteForm({
  className,
  ...props
}: CreateInviteFormProps) {
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
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
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
                        <SelectItem value="OWNER">Owner</SelectItem>
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
}
