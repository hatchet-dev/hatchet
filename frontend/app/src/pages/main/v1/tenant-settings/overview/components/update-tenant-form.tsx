import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useTenantDetails } from '@/hooks/use-tenant';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
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
    reset,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: tenant?.name,
    },
  });

  useEffect(() => {
    if (tenant?.name) {
      reset({ name: tenant.name });
    }
  }, [tenant?.name, reset]);

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;

  return (
    <form
      className={cn('flex flex-col items-end gap-1', className)}
      onSubmit={handleSubmit((d) => props.onSubmit(d))}
    >
      <div className="flex items-center gap-2">
        <Input
          {...register('name')}
          id="name"
          placeholder="My Tenant"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          className="w-[220px]"
          disabled={props.isLoading}
        />
        <Button disabled={props.isLoading} className="w-fit">
          {props.isLoading && <Spinner />}
          Save
        </Button>
      </div>
      {nameError && <div className="text-sm text-red-500">{nameError}</div>}
    </form>
  );
}
