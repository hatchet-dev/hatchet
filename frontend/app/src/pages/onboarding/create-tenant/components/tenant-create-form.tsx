import { OnboardingStepProps } from '../types';
import { Card, CardContent } from '@/components/v1/ui/card';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { TenantEnvironment } from '@/lib/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { cn } from '@/lib/utils';
import { CheckIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { Monitor, Settings, Rocket } from 'lucide-react';
import { useEffect, useMemo, useRef } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  name: z.string().min(4).max(32),
  environment: z.string().min(1),
});

interface TenantCreateFormProps
  extends OnboardingStepProps<{ name: string; environment: string }> {
  organizationList?: OrganizationForUserList;
  selectedOrganizationId?: string | null;
  onOrganizationChange?: (organizationId: string) => void;
  isCloudEnabled?: boolean;
}

export function TenantCreateForm({
  value,
  onChange,
  isLoading,
  fieldErrors,
  className,
  organizationList,
  selectedOrganizationId,
  onOrganizationChange,
  isCloudEnabled,
}: TenantCreateFormProps) {
  const {
    register,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      environment: 'development',
    },
  });

  const hasSetInitialDefault = useRef(false);

  const getEnvironmentName = (environment: string | undefined): string => {
    switch (environment) {
      case TenantEnvironment.Local:
        return 'local';
      case TenantEnvironment.Development:
        return 'dev';
      case TenantEnvironment.Production:
        return 'prod';
      default: {
        // Exhaustiveness check: this should never be reached if all cases are handled
        const exhaustiveCheck: never = environment as never;
        void exhaustiveCheck;
        return 'dev'; // Default to dev if no environment selected
      }
    }
  };

  const emptyState = useMemo(() => {
    // Use simple environment-based names
    const currentEnvironment = value?.environment || 'development';
    return getEnvironmentName(currentEnvironment);
  }, [value?.environment]);

  // Update form values when parent value changes
  useEffect(() => {
    const nameValue = value?.name ?? '';
    const environmentValue = value?.environment || 'development';

    setValue('name', nameValue);
    setValue('environment', environmentValue);

    // Set the generated name only once on initial load when no name is provided
    if (!hasSetInitialDefault.current && !value?.name && emptyState) {
      hasSetInitialDefault.current = true;
      onChange({
        name: emptyState,
        environment: environmentValue,
      });
    }
  }, [value, setValue, emptyState, onChange]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;

  const environmentOptions = [
    {
      value: 'local',
      label: 'Local Dev',
      icon: Monitor,
      description: 'Testing and development on your local machine',
    },
    {
      value: 'development',
      label: 'Development',
      icon: Settings,
      description: 'Shared development environment or staging',
    },
    {
      value: 'production',
      label: 'Production',
      icon: Rocket,
      description: 'Live production environment serving real users',
    },
  ];

  const handleEnvironmentChange = (selectedEnvironment: string) => {
    const updatedName = getEnvironmentName(selectedEnvironment);

    setValue('environment', selectedEnvironment);
    setValue('name', updatedName);

    onChange({
      name: updatedName,
      environment: selectedEnvironment,
    });
  };

  return (
    <div className={cn('grid gap-6', className)}>
      <div className="grid gap-4">
        {isCloudEnabled && organizationList && (
          <div className="grid gap-2">
            <Label>Organization</Label>
            <div className="text-sm text-gray-700 dark:text-gray-300">
              Select the organization to add this tenant to.
            </div>
            <Select
              value={selectedOrganizationId || undefined}
              onValueChange={onOrganizationChange}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select an organization..." />
              </SelectTrigger>
              <SelectContent>
                {organizationList.rows
                  ?.filter((org) => org.isOwner)
                  .map((org) => (
                    <SelectItem key={org.metadata.id} value={org.metadata.id}>
                      {org.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
            {fieldErrors?.organizationId && (
              <div className="text-sm text-red-500">
                {fieldErrors.organizationId}
              </div>
            )}
          </div>
        )}
        <div className="grid gap-2">
          <Label>Environment Type</Label>
          <div className="text-sm text-gray-700 dark:text-gray-300">
            You can add new tenants for different environments later.
          </div>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            {environmentOptions.map((option) => {
              const Icon = option.icon;
              const isSelected =
                (value?.environment || 'development') === option.value;

              return (
                <Card
                  key={option.value}
                  onClick={() => handleEnvironmentChange(option.value)}
                  className={`cursor-pointer transition-all hover:shadow-md ${
                    isSelected
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                      : 'hover:border-gray-300 dark:hover:border-gray-600'
                  }`}
                >
                  <CardContent className="flex flex-col items-center space-y-2 p-4 text-center">
                    <Icon
                      className={`h-6 w-6 ${isSelected ? 'text-blue-600 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'}`}
                    />
                    <div className="space-y-1">
                      <div className="text-sm font-medium">{option.label}</div>
                      <div className="text-xs text-gray-600 dark:text-gray-400">
                        {option.description}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>

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
                environment: value?.environment || 'development',
              });
            }}
          />
          {nameError && <div className="text-sm text-red-500">{nameError}</div>}
        </div>

        {/* Summary Section */}
        {isCloudEnabled && selectedOrganizationId && organizationList?.rows && (
          <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950/20">
            <div className="flex items-start gap-3">
              <div className="mt-0.5 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-blue-500">
                <CheckIcon className="size-3 text-white" />
              </div>
              <div className="flex-1">
                <h4 className="mb-1 text-sm font-medium text-blue-900 dark:text-blue-100">
                  Tenant Creation Summary
                </h4>
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  This tenant will be created in the{' '}
                  <span className="font-medium">
                    {
                      organizationList.rows.find(
                        (org) => org.metadata.id === selectedOrganizationId,
                      )?.name
                    }
                  </span>{' '}
                  organization.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
