import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { Monitor, Settings, Rocket } from 'lucide-react';
import { CheckIcon } from '@heroicons/react/24/outline';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useEffect, useMemo } from 'react';
import { OnboardingStepProps } from '../types';
import { useQuery } from '@tanstack/react-query';
import api, { TenantEnvironment } from '@/lib/api';
import freeEmailDomains from '@/lib/free-email-domains.json';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';

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
    const nameValue = value?.name || emptyState;
    const environmentValue = value?.environment || 'development';

    setValue('name', nameValue);
    setValue('environment', environmentValue);

    // Also update the parent if we're using the generated name
    if (!value?.name && emptyState && nameValue) {
      onChange({
        name: nameValue,
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
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
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
                  <CardContent className="p-4 flex flex-col items-center text-center space-y-2">
                    <Icon
                      className={`w-6 h-6 ${isSelected ? 'text-blue-600 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'}`}
                    />
                    <div className="space-y-1">
                      <div className="font-medium text-sm">{option.label}</div>
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
          <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0 w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center mt-0.5">
                <CheckIcon className="w-3 h-3 text-white" />
              </div>
              <div className="flex-1">
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-1">
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
