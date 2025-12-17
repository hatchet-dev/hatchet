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
import api, { TenantEnvironment } from '@/lib/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import freeEmailDomains from '@/lib/free-email-domains.json';
import { cn } from '@/lib/utils';
import { CheckIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
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
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      environment: 'development',
    },
  });

  const hasSetInitialDefault = useRef(false);

  const getEnvironmentPostfix = (environment: string | undefined): string => {
    switch (environment) {
      case TenantEnvironment.Local:
        return '-local';
      case TenantEnvironment.Development:
        return '-dev';
      case TenantEnvironment.Production:
        return '-prod';
      default: {
        // Exhaustiveness check: this should never be reached if all cases are handled
        const exhaustiveCheck: never = environment as never;
        void exhaustiveCheck;
        return '-dev'; // Default to dev if no environment selected
      }
    }
  };

  const hasEnvironmentPostfix = (name: string): boolean => {
    return (
      name.endsWith('-local') || name.endsWith('-dev') || name.endsWith('-prod')
    );
  };

  const removeEnvironmentPostfix = (name: string): string => {
    if (name.endsWith('-local')) {
      return name.slice(0, -6);
    }
    if (name.endsWith('-dev')) {
      return name.slice(0, -4);
    }
    if (name.endsWith('-prod')) {
      return name.slice(0, -5);
    }
    return name;
  };

  const updateNameWithEnvironment = (
    currentName: string,
    environment: string,
  ): string => {
    const baseName = hasEnvironmentPostfix(currentName)
      ? removeEnvironmentPostfix(currentName)
      : currentName;

    return baseName + getEnvironmentPostfix(environment);
  };

  const emptyState = useMemo(() => {
    if (!user.data?.email) {
      return '';
    }

    const email = user.data.email;
    const [localPart, domain] = email.split('@');

    let baseName = '';
    if (freeEmailDomains.includes(domain?.toLowerCase())) {
      baseName = localPart;
    } else {
      // For business emails, use the domain without the TLD
      const domainParts = domain?.split('.');
      baseName = domainParts?.[0] || localPart;
    }

    // Add environment-specific postfix using current environment
    const currentEnvironment = value?.environment || 'development';
    return `${baseName}${getEnvironmentPostfix(currentEnvironment)}`;
  }, [user.data?.email, value?.environment]);

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
    const currentName = value?.name || '';
    const updatedName = currentName
      ? updateNameWithEnvironment(currentName, selectedEnvironment)
      : '';

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
