import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/ui/icons';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useEffect, useState } from 'react';

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
  //   const [isSlugModified, setIsSlugModified] = useState(false);
  const [isSlugSuffixed, setIsSlugSuffixed] = useState(false);

  const {
    register,
    handleSubmit,
    setValue,
    getValues,
    watch,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      switch (name) {
        case 'name':
          if (!isSlugSuffixed) {
            const slug = value.name
              ?.toLowerCase()
              .replace(/[^a-z0-9-]/g, '-')
              .replace(/-+/g, '-')
              .replace(/^-|-$/g, '');

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
  }, [isSlugSuffixed, setValue, watch]);

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
            <div className="text-sm text-muted-foreground">
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
              onBlur={() => {
                // add a random suffix to the slug if it's not modified
                if (!isSlugSuffixed) {
                  const newSlug =
                    getValues('slug') +
                    '-' +
                    Math.random().toString(36).substr(2, 5);
                  setValue('slug', newSlug);
                  setIsSlugSuffixed(true);
                }
              }}
            />
            {nameError && (
              <div className="text-sm text-red-500">{nameError}</div>
            )}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="name">Slug</Label>
            <div className="text-sm text-muted-foreground">
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
          <Button disabled={props.isLoading || !isSlugSuffixed}>
            {props.isLoading && (
              <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
            )}
            Create
          </Button>
        </div>
      </form>
    </div>
  );
}
