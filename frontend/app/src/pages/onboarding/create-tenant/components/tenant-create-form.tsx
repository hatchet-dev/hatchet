import { OnboardingStepProps } from '../types';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import api from '@/lib/api';
import { OrganizationForUserList } from '@/lib/api/generated/cloud/data-contracts';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { LogOut, Plus } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  name: z.string().min(1).max(32),
  referralSource: z.string().optional(),
});

interface TenantCreateFormProps
  extends OnboardingStepProps<{
    name: string;
    environment: string;
    referralSource?: string;
  }> {
  organizationList?: OrganizationForUserList;
  selectedOrganizationId?: string | null;
  onOrganizationChange?: (organizationId: string) => void;
  isCloudEnabled?: boolean;
  existingTenantNames?: string[];
}

export function TenantCreateForm({
  value,
  onChange,
  onNext,
  isLoading,
  fieldErrors,
  className,
  organizationList,
  selectedOrganizationId,
  onOrganizationChange,
  isCloudEnabled,
  existingTenantNames,
}: TenantCreateFormProps) {
  const { currentUser } = useCurrentUser();
  const navigate = useNavigate();
  const { handleCreateOrganization, createOrganizationLoading } =
    useOrganizations();

  const [showCreateOrgModal, setShowCreateOrgModal] = useState(false);
  const [orgName, setOrgName] = useState('');

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
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
      referralSource: '',
    },
  });

  const hasSetInitialDefault = useRef(false);

  const defaultTenantName = useMemo(() => {
    if (!currentUser) {
      return '';
    }
    const rawName = currentUser.name?.trim();
    const emailPrefix = currentUser.email?.split('@')[0]?.trim();
    const base = rawName || emailPrefix || '';

    const slugBase = base
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');

    if (!slugBase) {
      return '';
    }

    const candidate = `${slugBase}-development`;
    const existing = (existingTenantNames ?? []).map((n) => n.toLowerCase());
    if (existing.includes(candidate.toLowerCase())) {
      return '';
    }

    return candidate;
  }, [currentUser, existingTenantNames]);

  // Update form values when parent value changes
  useEffect(() => {
    const nameValue = value?.name ?? '';
    const referralSourceValue = value?.referralSource ?? '';

    setValue('name', nameValue);
    setValue('referralSource', referralSourceValue);

    // Set the generated name only once on initial load when no name is provided
    if (!hasSetInitialDefault.current && !value?.name && defaultTenantName) {
      hasSetInitialDefault.current = true;
      setValue('name', defaultTenantName);
      onChange({
        name: defaultTenantName,
        environment: value?.environment || 'development',
        referralSource: referralSourceValue,
      });
    }
  }, [value, setValue, defaultTenantName, onChange]);

  const nameError = errors.name?.message?.toString() || fieldErrors?.name;
  const hasExistingTenants = (existingTenantNames?.length ?? 0) > 0;

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
              onValueChange={(nextValue) => {
                if (nextValue === '__create_organization__') {
                  setShowCreateOrgModal(true);
                  return;
                }

                onOrganizationChange?.(nextValue);
              }}
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
                <SelectItem value="__create_organization__">
                  <div className="flex items-center gap-2">
                    <Plus className="size-4" />
                    Create organization…
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
            {fieldErrors?.organizationId && (
              <div className="text-sm text-red-500">
                {fieldErrors.organizationId}
              </div>
            )}
          </div>
        )}

        <div className="grid gap-3">
          <Label htmlFor="name">Tenant name</Label>
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
                referralSource: value?.referralSource,
              });
            }}
          />
          <div className="text-xs text-gray-700 dark:text-gray-300">
            You can always rename your tenant later.
          </div>
          {nameError && <div className="text-sm text-red-500">{nameError}</div>}
        </div>

        {!hasExistingTenants && (
          <div className="grid gap-3">
            <Label htmlFor="referral_source">
              Where did you hear about us? (optional)
            </Label>
            <Input
              {...register('referralSource')}
              id="referral_source"
              placeholder="e.g. Twitter, LinkedIn, etc."
              type="text"
              autoCapitalize="none"
              autoCorrect="off"
              disabled={isLoading}
              onChange={(e) => {
                setValue('referralSource', e.target.value);
                onChange({
                  name: value?.name || '',
                  environment: value?.environment || 'development',
                  referralSource: e.target.value,
                });
              }}
            />
          </div>
        )}

        {/* Submit Button */}
        <Button
          variant="default"
          className="w-full"
          onClick={onNext}
          disabled={isLoading || !value?.name || value.name.length < 1}
        >
          {isLoading ? (
            <div className="flex items-center gap-2">
              <div className="size-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              Creating...
            </div>
          ) : (
            'Create Tenant'
          )}
        </Button>

        {/* Help Section */}
        <div className="text-center text-sm text-muted-foreground">
          Have questions?{' '}
          <a
            href="https://docs.hatchet.run"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            Visit documentation
          </a>{' '}
          or{' '}
          <a
            href="https://hatchet.run/discord"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            join our Discord
          </a>
          .
        </div>

        <div className="flex justify-center">
          <Button
            leftIcon={<LogOut className="size-4" />}
            onClick={() => logoutMutation.mutate()}
            size="sm"
            variant="ghost"
            disabled={isLoading || logoutMutation.isPending}
          >
            Log out
          </Button>
        </div>
      </div>

      <Dialog open={showCreateOrgModal} onOpenChange={setShowCreateOrgModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create organization</DialogTitle>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label htmlFor="org-name">Organization name</Label>
            <Input
              id="org-name"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              placeholder="Enter organization name"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && orgName.trim()) {
                  handleCreateOrganization(orgName.trim(), (organizationId) => {
                    setShowCreateOrgModal(false);
                    setOrgName('');
                    onOrganizationChange?.(organizationId);
                  });
                }
              }}
              disabled={createOrganizationLoading}
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowCreateOrgModal(false);
                setOrgName('');
              }}
              disabled={createOrganizationLoading}
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!orgName.trim()) {
                  return;
                }
                handleCreateOrganization(orgName.trim(), (organizationId) => {
                  setShowCreateOrgModal(false);
                  setOrgName('');
                  onOrganizationChange?.(organizationId);
                });
              }}
              disabled={!orgName.trim() || createOrganizationLoading}
            >
              {createOrganizationLoading ? 'Creating…' : 'Create'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
