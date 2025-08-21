import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useEffect } from 'react';
import { OnboardingStepProps } from '../types';

const schema = z.object({
  name: z.string().min(4).max(32),
  slug: z.string().min(4).max(32),
});

interface TenantCreateFormProps
  extends OnboardingStepProps<{ name: string; slug: string }> {}

export function TenantCreateForm({
  value,
  onChange,
  isLoading,
  fieldErrors,
  className,
}: TenantCreateFormProps) {
  const {
    register,
    setValue,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: value?.name || '',
      slug: value?.slug || '',
    },
  });

  useEffect(() => {
    const subscription = watch((watchValue, { name }) => {
      switch (name) {
        case 'name':
          if (watchValue.name) {
            const slug =
              watchValue.name
                ?.toLowerCase()
                .replace(/[^a-z0-9-]/g, '-')
                .replace(/-+/g, '-')
                .replace(/^-|-$/g, '') +
              '-' +
              Math.random().toString(36).substr(2, 5);

            if (slug) {
              setValue('slug', slug);
              onChange({ name: watchValue.name, slug });
            }
          }
          break;
        case 'slug':
          if (watchValue.slug) {
            onChange({ name: watchValue.name || '', slug: watchValue.slug });
          }
          break;
      }
    });

    return () => subscription.unsubscribe();
  }, [setValue, watch, onChange]);

  // Update form values when parent value changes
  useEffect(() => {
    if (value) {
      setValue('name', value.name || '');
      setValue('slug', value.slug || '');
    }
  }, [value, setValue]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;

  const slugError = errors.slug?.message?.toString() || fieldErrors?.slug;

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
                slug: e.target.value ? value?.slug || '' : '',
              });

              // if value is unset, reset the slug
              if (!e.target.value) {
                setValue('slug', '');
              }
            }}
          />
          {nameError && <div className="text-sm text-red-500">{nameError}</div>}
        </div>
        <div className="grid gap-2">
          <Label htmlFor="name">Slug</Label>
          <div className="text-sm text-gray-700 dark:text-gray-300">
            A URI-friendly identifier for your tenant.
          </div>
          <Input
            {...register('slug')}
            id="slug"
            placeholder="my-awesome-tenant-123456"
            type="name"
            autoCapitalize="none"
            autoCorrect="off"
            disabled={isLoading}
            spellCheck={false}
            onChange={(e) => {
              setValue('slug', e.target.value);
              onChange({
                name: value?.name || '',
                slug: e.target.value,
              });
            }}
          />
          {slugError && <div className="text-sm text-red-500">{slugError}</div>}
        </div>
      </div>
    </div>
  );
}
