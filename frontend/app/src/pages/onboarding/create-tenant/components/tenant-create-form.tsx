import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useEffect, useMemo } from 'react';
import freeEmailDomains from 'free-email-domains';
import { OnboardingStepProps } from '../types';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';

const schema = z.object({
  name: z.string().min(4).max(32),
});

interface TenantCreateFormProps extends OnboardingStepProps<{ name: string }> {}

export function TenantCreateForm({
  value,
  onChange,
  isLoading,
  fieldErrors,
  className,
}: TenantCreateFormProps) {
  const user = useQuery({
    queryKey: ['user:get:current'],
    retry: false,
    queryFn: async () => {
      const res = await api.userGetCurrent();
      return res.data;
    },
  });

  const {
    register,
    setValue,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: value?.name || '',
    },
  });

  useEffect(() => {
    const subscription = watch((watchValue, { name }) => {
      if (name === 'name' && watchValue.name) {
        onChange({ name: watchValue.name });
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, onChange]);

  const emptyState = useMemo(() => {
    if (!user.data?.email) {
      return '';
    }

    const email = user.data.email;
    const [localPart, domain] = email.split('@');

    if (freeEmailDomains.includes(domain?.toLowerCase())) {
      return localPart;
    } else {
      // For business emails, use the domain without the TLD
      const domainParts = domain?.split('.');
      return domainParts?.[0] || localPart;
    }
  }, [user.data?.email]);

  // Update form values when parent value changes
  useEffect(() => {
    if (value) {
      setValue('name', value.name || `${emptyState}-dev`);
    }
  }, [value, setValue, emptyState]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;

  return (
    <div className={cn('grid gap-6', className)}>
      <div className="grid gap-4">
        <div className="grid gap-2">
          <Label htmlFor="name">Name</Label>
          <div className="text-sm text-gray-700 dark:text-gray-300">
            A display name for your tenant.
          </div>
          <Input
            {...register('name')}
            id="name"
            placeholder="My Awesome Tenant"
            type="name"
            autoCapitalize="none"
            autoCorrect="off"
            disabled={isLoading}
            spellCheck={false}
            onChange={(e) => {
              setValue('name', e.target.value);
              onChange({
                name: e.target.value,
              });
            }}
          />
          {nameError && <div className="text-sm text-red-500">{nameError}</div>}
        </div>
      </div>
    </div>
  );
}
