import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useTenantDetails } from '@/hooks/use-tenant';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  name: z.string().max(255).min(1),
});

interface UpdateTenantSettingsProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function UpdateTenantForm({
  className,
  ...props
}: UpdateTenantSettingsProps) {
  const { tenant } = useTenantDetails();

  const {
    handleSubmit,
    register,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: tenant?.name,
    },
  });

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.role;

  return (
    <div className={cn('grid gap-6', className)}>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit(d);
        })}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="email">Name</Label>
            <Input
              {...register('name')}
              id="name"
              placeholder="My Tenant"
              type="name"
              autoCapitalize="none"
              autoCorrect="off"
              className="min-w-[300px]"
              disabled={props.isLoading}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
          </div>
          <Button disabled={props.isLoading} className="w-fit">
            {props.isLoading && <Spinner />}
            Change Name
          </Button>
        </div>
      </form>
    </div>
  );
}
