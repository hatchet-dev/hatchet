import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/loading.tsx';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useEffect } from 'react';

const schema = z.object({
  name: z.string().min(4).max(32),
  slug: z.string().min(4).max(32),
});

interface TenantCreateFormProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function TenantCreateForm({
  className,
  ...props
}: TenantCreateFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      switch (name) {
        case 'name':
          if (value.name) {
            const slug =
              value.name
                ?.toLowerCase()
                .replace(/[^a-z0-9-]/g, '-')
                .replace(/-+/g, '-')
                .replace(/^-|-$/g, '') +
              '-' +
              Math.random().toString(36).substr(2, 5);

            if (slug) {
              setValue('slug', slug);
            }
          }

          break;
        case 'slug':
          break;
      }
    });

    return () => subscription.unsubscribe();
  }, [setValue, watch]);

  const nameError =
    errors.name?.message?.toString() || props.fieldErrors?.email;

  const slugError = errors.slug?.message?.toString() || props.fieldErrors?.slug;

  return (
    <div className={cn('grid gap-6', className)}>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit(d);
        })}
      >
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
              disabled={props.isLoading}
              spellCheck={false}
              onChange={(e) => {
                setValue('name', e.target.value);

                // if value is unset, reset the slug
                if (!e.target.value) {
                  setValue('slug', '');
                }
              }}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
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
              disabled={props.isLoading}
              spellCheck={false}
            />
            {slugError && (
              <div className="text-sm text-red-500">{slugError}</div>
            )}
          </div>
          <Button disabled={props.isLoading}>
            {props.isLoading && <Spinner />}
            Create
          </Button>
        </div>
      </form>
    </div>
  );
}
