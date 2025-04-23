import useTenant from '@/next/hooks/use-tenant';
import { useForm } from 'react-hook-form';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import { Label } from '@/next/components/ui/label';
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import useUser from '@/next/hooks/use-user';
import { useCallback } from 'react';
import { FaDice } from 'react-icons/fa';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import {
  generateRandomName,
  validateTenantName,
} from '@/next/lib/utils/name-generator';

interface TenantFormValues {
  name: string;
}

export default function OnboardingNewPage() {
  const { create: createTenant, setTenant } = useTenant();
  const { data: user } = useUser();
  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
    setValue,
  } = useForm<TenantFormValues>();

  const onSubmit = async (data: TenantFormValues) => {
    try {
      createTenant.mutate(data.name, {
        onSuccess: (tenant) => {
          setTenant(tenant.metadata.id);
          window.location.href = '/?tenant=' + tenant.metadata.id;
        },
        onError: (error) => {
          setError('name', {
            type: 'server',
            message: error.message || 'Failed to create workspace',
          });
        },
      });
    } catch (error) {
      // Fallback error handling
      setError('name', {
        type: 'server',
        message: 'Failed to create tenant',
      });
    }
  };

  const handleGenerateRandomName = useCallback(() => {
    setValue('name', generateRandomName());
  }, [setValue]);

  return (
    <div className="flex justify-center items-start py-12 px-4">
      <div className="max-w-xl w-full pt-12">
        <Card>
          <CardHeader>
            <CardTitle>Create New Tenant</CardTitle>
            <CardDescription>
              Tenants are isolated environments that are used to organize your
              workloads.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form id="create-tenant-form" onSubmit={handleSubmit(onSubmit)}>
              <div className="space-y-2">
                <Label htmlFor="name">Friendly Name</Label>
                <div className="relative">
                  {user ? (
                    <Input
                      id="tenant-name"
                      autoComplete="off"
                      data-lpignore="true"
                      data-form-type="other"
                      autoCapitalize="off"
                      autoCorrect="off"
                      spellCheck="false"
                      {...register('name', {
                        validate: validateTenantName,
                      })}
                      minLength={5}
                      defaultValue={`${user?.name?.replace(' ', '-')}-dev`}
                      disabled={createTenant.isPending}
                      aria-invalid={errors.name ? 'true' : 'false'}
                      autoFocus
                    />
                  ) : (
                    <Input disabled={true} />
                  )}
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="absolute right-0 top-0 h-full px-3 py-2 text-muted-foreground hover:text-foreground"
                          onClick={handleGenerateRandomName}
                          disabled={createTenant.isPending}
                        >
                          <FaDice className="h-4 w-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Generate random name</p>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
                {errors.name && (
                  <p className="text-sm text-destructive mt-1">
                    {errors.name.message}
                  </p>
                )}
              </div>
            </form>
          </CardContent>
          <CardFooter className="flex justify-between gap-2">
            <DocsButton doc={docs.home.environments} size="icon" />
            <Button
              type="submit"
              form="create-tenant-form"
              loading={createTenant.isPending}
              disabled={createTenant.isPending}
            >
              Create Workspace
            </Button>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
