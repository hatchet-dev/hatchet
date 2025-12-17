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
import { TenantMember, TenantMemberRole } from '@/lib/api';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  role: z.nativeEnum(TenantMemberRole),
});

interface UpdateMemberFormProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  member: TenantMember;
  isCloudEnabled?: boolean;
}

export function UpdateMemberForm({
  className,
  ...props
}: UpdateMemberFormProps) {
  const {
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      role: props.member.role,
    },
  });

  const roleError = errors.role?.message?.toString() || props.fieldErrors?.role;

  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Update member role</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
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
              Update member
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
