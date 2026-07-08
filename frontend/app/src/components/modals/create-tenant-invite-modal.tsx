import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useOrganizations } from '@/hooks/use-organizations';
import { TenantMember, TenantMemberRole } from '@/lib/api';
import { TenantStatusType } from '@/lib/api/generated/cloud/data-contracts';
import {
  CreateTenantInviteRequest,
  TenantInvite,
} from '@/lib/api/generated/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { CheckIcon } from '@heroicons/react/24/outline';
import { zodResolver } from '@hookform/resolvers/zod';
import { CaretSortIcon } from '@radix-ui/react-icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

function MemberEmailSelect({
  value,
  onChange,
  options,
  loading,
}: {
  value?: string;
  onChange: (email: string) => void;
  options: string[];
  loading?: boolean;
}) {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
        >
          <span className={cn('truncate', !value && 'text-muted-foreground')}>
            {value || (loading ? 'Loading members...' : 'Select a member...')}
          </span>
          <CaretSortIcon className="ml-2 size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[--radix-popover-trigger-width] p-0"
        align="start"
      >
        <Command>
          <CommandInput placeholder="Search members..." />
          <CommandList>
            <CommandEmpty>No members found.</CommandEmpty>
            {options.map((email) => (
              <CommandItem
                key={email}
                value={email}
                onSelect={() => {
                  onChange(email);
                  setOpen(false);
                }}
                className="cursor-pointer"
              >
                <span className="truncate">{email}</span>
                <CheckIcon
                  className={cn(
                    'ml-auto size-4 shrink-0',
                    value === email ? 'opacity-100' : 'opacity-0',
                  )}
                />
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

type TenantOption = {
  id: string;
  label: string;
};

type CreateTenantInviteFormProps = {
  className?: string;
  onSubmit: (opts: {
    email: string;
    role: TenantMemberRole;
    tenantId?: string;
  }) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  formErrors?: string[];
  isCloudEnabled?: boolean;
  defaultEmail?: string;
  defaultTenantId?: string;
  tenantOptions?: TenantOption[];
  tenantOptionsLoading?: boolean;
  // When set, the email field is a searchable dropdown of these organization
  // members instead of a free-text input.
  emailOptions?: string[];
  emailOptionsLoading?: boolean;
  organizationId?: string;
  onTenantChange?: (tenantId: string) => void;
  onNavigateAway?: () => void;
};

const CreateTenantInviteForm = ({
  className,
  ...props
}: CreateTenantInviteFormProps) => {
  const showTenantSelect = props.tenantOptions !== undefined;

  const schema = useMemo(() => {
    const availableRoles = props.isCloudEnabled
      ? [TenantMemberRole.ADMIN, TenantMemberRole.MEMBER]
      : [
          TenantMemberRole.OWNER,
          TenantMemberRole.ADMIN,
          TenantMemberRole.MEMBER,
        ];
    return z.object({
      email: z.string().email('Invalid email address'),
      role: z.enum(availableRoles as [TenantMemberRole, ...TenantMemberRole[]]),
      tenantId: showTenantSelect
        ? z.string({ required_error: 'Select a tenant' }).min(1)
        : z.string().optional(),
    });
  }, [props.isCloudEnabled, showTenantSelect]);

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: props.defaultEmail ?? '',
      tenantId: props.defaultTenantId,
    },
  });

  const emailError =
    errors.email?.message?.toString() || props.fieldErrors?.email;

  const roleError = errors.role?.message?.toString() || props.fieldErrors?.role;

  const tenantError = errors.tenantId?.message?.toString();

  return (
    <DialogContent className="max-w-lg">
      <DialogHeader>
        <DialogTitle>Add member to tenant</DialogTitle>
        <DialogDescription>
          {showTenantSelect
            ? 'Attach a user to a tenant directly. For larger teams, we recommend using user groups to manage user access to tenants.'
            : 'Invite a user to this tenant by email.'}
        </DialogDescription>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <InlineError errors={props.formErrors ?? []} />
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              {props.emailOptions !== undefined ? (
                <>
                  {props.emailOptions.length === 0 &&
                  !props.emailOptionsLoading ? (
                    <Input
                      disabled
                      placeholder="All organization members are already in this tenant"
                    />
                  ) : (
                    <Controller
                      control={control}
                      name="email"
                      render={({ field }) => (
                        <MemberEmailSelect
                          value={field.value}
                          onChange={field.onChange}
                          options={props.emailOptions ?? []}
                          loading={props.emailOptionsLoading}
                        />
                      )}
                    />
                  )}
                  {props.organizationId && (
                    <p className="text-xs text-muted-foreground">
                      Not seeing your teammates?{' '}
                      <Link
                        to={appRoutes.organizationTeamRoute.to}
                        params={{ organization: props.organizationId }}
                        onClick={props.onNavigateAway}
                        className="text-primary underline-offset-4 hover:underline"
                      >
                        Manage organization members
                      </Link>
                    </p>
                  )}
                </>
              ) : (
                <Input
                  {...register('email')}
                  id="email"
                  placeholder="name@example.com"
                  type="email"
                  autoCapitalize="none"
                  autoComplete="email"
                  autoCorrect="off"
                  disabled={props.isLoading}
                />
              )}
              {emailError && (
                <div className="text-sm text-red-500">{emailError}</div>
              )}
            </div>
            {showTenantSelect && (
              <div className="grid gap-2">
                <Label htmlFor="tenant">Tenant</Label>
                <Controller
                  control={control}
                  name="tenantId"
                  render={({ field }) => {
                    return (
                      <Select
                        onValueChange={(value) => {
                          field.onChange(value);
                          props.onTenantChange?.(value);
                        }}
                        value={field.value}
                        disabled={props.tenantOptionsLoading}
                      >
                        <SelectTrigger className="w-full focus:ring-0 focus-visible:ring-0">
                          <SelectValue
                            id="tenant"
                            placeholder={
                              props.tenantOptionsLoading
                                ? 'Loading tenants...'
                                : 'Tenant...'
                            }
                          />
                        </SelectTrigger>
                        <SelectContent>
                          {(props.tenantOptions ?? []).map((tenant) => (
                            <SelectItem key={tenant.id} value={tenant.id}>
                              {tenant.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    );
                  }}
                />
                {tenantError && (
                  <div className="text-sm text-red-500">{tenantError}</div>
                )}
              </div>
            )}
            <div className="grid gap-2">
              <Label htmlFor="role">Role</Label>
              <Controller
                control={control}
                name="role"
                render={({ field }) => {
                  return (
                    <Select onValueChange={field.onChange} value={field.value}>
                      <SelectTrigger className="w-[180px] focus:ring-0 focus-visible:ring-0">
                        <SelectValue id="role" placeholder="Role..." />
                      </SelectTrigger>
                      <SelectContent>
                        {!props.isCloudEnabled && (
                          <SelectItem value="OWNER">Owner</SelectItem>
                        )}
                        <SelectItem value="ADMIN">Admin</SelectItem>
                        <SelectItem value="MEMBER">Member</SelectItem>
                      </SelectContent>
                    </Select>
                  );
                }}
              />
              {roleError && (
                <div className="text-sm text-red-500">{roleError}</div>
              )}
            </div>
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Invite user
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
};

export const CreateTenantInviteModal = ({
  tenantId,
  organizationId,
  defaultEmail,
  onClose,
  onCreated,
}: {
  tenantId?: string;
  organizationId?: string;
  defaultEmail?: string;
  onClose: () => void;
  onCreated: (tenantId: string, invite: TenantInvite) => void;
}) => {
  const { isCloudEnabled } = useOrganizations();
  // `fieldErrors` is only for the client-side duplicate-member guard below.
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formErrors, setFormErrors] = useState<string[]>([]);
  // Route API errors to an inline banner rather than a toast, which would
  // render behind the modal overlay.
  const { handleApiError } = useApiError({
    setErrors: setFormErrors,
  });

  const queryClient = useQueryClient();

  // Whenever an organization is known, the user picks (or confirms) one of
  // its tenants — pre-selected when opened from a tenant-scoped page. Without
  // an organization (OSS), the invite is fixed to the current tenant.
  const needsTenantSelect = !!organizationId;

  const orgApi = useOrganizationApi();
  const orgQuery = useQuery({
    ...orgApi.organizationGetQuery(organizationId ?? ''),
    enabled: needsTenantSelect,
  });

  // Track the tenant selected in the form so the member dropdown can exclude
  // users who are already members of it.
  const [watchedTenantId, setWatchedTenantId] = useState(tenantId);
  const { tenantMemberListQuery } = useTenantApi();
  const tenantMembersQuery = useQuery({
    ...tenantMemberListQuery(watchedTenantId ?? ''),
    enabled: needsTenantSelect && !!watchedTenantId,
  });

  const tenantMemberEmails = useMemo(
    () =>
      new Set(
        (tenantMembersQuery.data?.rows ?? []).map(
          (member: TenantMember) => member.user.email,
        ),
      ),
    [tenantMembersQuery.data?.rows],
  );

  // Org mode: the email field is a dropdown of org members who aren't already
  // in the selected tenant.
  const emailOptions = useMemo(() => {
    if (!needsTenantSelect) {
      return undefined;
    }

    return (orgQuery.data?.members ?? [])
      .map((member) => member.email)
      .filter((email) => !tenantMemberEmails.has(email));
  }, [needsTenantSelect, orgQuery.data?.members, tenantMemberEmails]);

  const tenantOptions = useMemo(() => {
    if (!needsTenantSelect) {
      return undefined;
    }
    return (orgQuery.data?.tenants ?? [])
      .filter((tenant) => tenant.status !== TenantStatusType.ARCHIVED)
      .map((tenant) => ({
        id: tenant.id,
        label: tenant.name ?? tenant.slug ?? tenant.id,
      }));
  }, [needsTenantSelect, orgQuery.data?.tenants]);

  const { tenantInviteCreateMutation } = useTenantApi();
  const createMutation = useMutation({
    mutationKey: ['tenant-invite:create'],
    mutationFn: async ({
      tenantId: inviteTenantId,
      data,
    }: {
      tenantId: string;
      data: CreateTenantInviteRequest;
    }) => {
      const invite =
        await tenantInviteCreateMutation(inviteTenantId).mutationFn(data);
      return { tenantId: inviteTenantId, invite };
    },
    onSuccess: ({ tenantId: inviteTenantId, invite }) => {
      queryClient.invalidateQueries({
        queryKey: ['tenant-invite:list', inviteTenantId],
      });
      onCreated(inviteTenantId, invite);
      onClose();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <CreateTenantInviteForm
        isLoading={createMutation.isPending}
        onSubmit={({ email, role, tenantId: selectedTenantId }) => {
          // Clear any prior errors before re-submitting.
          setFieldErrors({});
          setFormErrors([]);
          // When the tenant select is shown, the user's choice wins over the
          // pre-selected tenant.
          const inviteTenantId = needsTenantSelect
            ? selectedTenantId
            : tenantId;
          if (!inviteTenantId) {
            return;
          }
          // Guard against a stale selection when the tenant was changed after
          // picking a member who already belongs to the new tenant.
          if (needsTenantSelect && tenantMemberEmails.has(email)) {
            setFieldErrors({
              email: 'This user is already a member of the selected tenant.',
            });
            return;
          }
          createMutation.mutate({
            tenantId: inviteTenantId,
            data: { email, role },
          });
        }}
        fieldErrors={fieldErrors}
        formErrors={formErrors}
        isCloudEnabled={isCloudEnabled}
        defaultEmail={defaultEmail}
        defaultTenantId={needsTenantSelect ? tenantId : undefined}
        tenantOptions={tenantOptions}
        tenantOptionsLoading={needsTenantSelect && orgQuery.isLoading}
        emailOptions={emailOptions}
        emailOptionsLoading={orgQuery.isLoading || tenantMembersQuery.isLoading}
        organizationId={organizationId}
        onTenantChange={setWatchedTenantId}
        onNavigateAway={onClose}
      />
    </Dialog>
  );
};
