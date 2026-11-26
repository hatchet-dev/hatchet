import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/v1/ui/collapsible';
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { SecretCopier } from '@/components/v1/ui/secret-copier';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { CreateAPITokenRequest, queries } from '@/lib/api';
import {
  CapabilityCatalog,
  buildCatalog,
  grantedCapabilities,
  toRequestedCapabilities,
} from '@/lib/api-token-capabilities';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { CheckIcon, ChevronRightIcon, Cross2Icon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

const EXPIRES_IN_OPTS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

function presets(catalog: CapabilityCatalog) {
  return [
    {
      id: 'read',
      label: 'Total read access',
      description: 'Everything a tenant viewer can read.',
      capabilities: catalog.read,
    },
    {
      id: 'write',
      label: 'Total write access',
      description:
        'Running workers, triggering and managing runs, and changing the tenant.',
      capabilities: catalog.write,
    },
  ];
}

const schema = z.object({
  name: z.string().min(1).max(255),
  expiresIn: z.string().optional(),
  capabilities: z.array(z.string()).min(1, 'Select at least one capability'),
});

const unscopedSchema = schema.extend({ capabilities: z.array(z.string()) });

function toRequest(
  d: z.infer<typeof schema>,
  catalog: CapabilityCatalog,
): CreateAPITokenRequest {
  if (catalog.selectable.length === 0) {
    return { name: d.name, expiresIn: d.expiresIn };
  }

  return {
    name: d.name,
    expiresIn: d.expiresIn,
    capabilities: toRequestedCapabilities(d.capabilities, catalog),
  };
}

interface CreateTokenDialogProps {
  className?: string;
  token?: string;
  onSubmit: (opts: CreateAPITokenRequest) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateTokenDialog({ token, ...props }: CreateTokenDialogProps) {
  const { tenantId } = useCurrentTenantId();
  const capabilitiesQuery = useQuery(queries.tokens.capabilities(tenantId));
  const catalog = useMemo(
    () =>
      capabilitiesQuery.data
        ? buildCatalog(capabilitiesQuery.data.rows)
        : undefined,
    [capabilitiesQuery.data],
  );

  if (token) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          This is the only time we will show you this token. Make sure to copy
          it somewhere safe.
        </p>
        <SecretCopier
          secrets={{ HATCHET_CLIENT_TOKEN: token }}
          className="text-sm"
          maxWidth={'calc(700px - 4rem)'}
          copy
        />
      </DialogContent>
    );
  }

  // A shard older than the capabilities endpoint cannot scope tokens, so it gets the unscoped form,
  // whose tokens default to "all".
  if (capabilitiesQuery.isError) {
    return <CreateTokenForm catalog={buildCatalog([])} {...props} />;
  }

  if (!catalog) {
    return (
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Create a new API token</DialogTitle>
        </DialogHeader>
        <Spinner />
      </DialogContent>
    );
  }

  return <CreateTokenForm catalog={catalog} {...props} />;
}

// The form mounts once the catalog is loaded, so its defaults can select every capability.
function CreateTokenForm({
  className,
  catalog,
  ...props
}: Omit<CreateTokenDialogProps, 'token'> & { catalog: CapabilityCatalog }) {
  const scoped = catalog.selectable.length > 0;
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(scoped ? schema : unscopedSchema),
    defaultValues: { capabilities: catalog.selectable },
  });
  const [review, setReview] = useState<z.infer<typeof schema>>();

  // The form stays mounted under the review step, so going back keeps what was entered.
  if (review) {
    return (
      <ReviewCapabilities
        catalog={catalog}
        values={review}
        isLoading={props.isLoading}
        fieldErrors={props.fieldErrors}
        onBack={() => setReview(undefined)}
        onAccept={() => props.onSubmit(toRequest(review, catalog))}
      />
    );
  }

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;
  const capabilitiesError =
    errors.capabilities?.message?.toString() || props.fieldErrors?.capabilities;

  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Create a new API token</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) =>
            scoped ? setReview(d) : props.onSubmit(toRequest(d, catalog)),
          )}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="api-token-name">Name</Label>
              <Input
                {...register('name')}
                id="api-token-name"
                placeholder="My Token"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="expiresIn">Expires In</Label>
              <Controller
                control={control}
                defaultValue={EXPIRES_IN_OPTS['100 years']}
                name="expiresIn"
                render={({ field }) => {
                  return (
                    <Select onValueChange={field.onChange} {...field}>
                      <SelectTrigger id="expiresIn">
                        <SelectValue
                          id="expiresInSelected"
                          placeholder="Select a duration"
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {Object.entries(EXPIRES_IN_OPTS).map(
                          ([label, value]) => (
                            <SelectItem key={value} value={value}>
                              {label}
                            </SelectItem>
                          ),
                        )}
                      </SelectContent>
                    </Select>
                  );
                }}
              />
            </div>
            {scoped && (
              <Controller
                control={control}
                name="capabilities"
                render={({ field }) => (
                  <CapabilityPicker
                    catalog={catalog}
                    value={field.value}
                    onChange={field.onChange}
                    disabled={props.isLoading}
                  />
                )}
              />
            )}
            {capabilitiesError && (
              <div className="text-sm text-red-500">{capabilitiesError}</div>
            )}
            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Create token
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}

function selectionState(
  names: string[],
  selected: string[],
): boolean | 'indeterminate' {
  const count = names.filter((n) => selected.includes(n)).length;

  if (count === 0) {
    return false;
  }

  return count === names.length ? true : 'indeterminate';
}

function CapabilityPicker({
  catalog,
  value,
  onChange,
  disabled,
}: {
  catalog: CapabilityCatalog;
  value: string[];
  onChange: (capabilities: string[]) => void;
  disabled: boolean;
}) {
  const toggle = (names: string[], on: boolean) =>
    onChange(
      on
        ? [...new Set([...value, ...names])]
        : value.filter((c) => !names.includes(c)),
    );

  return (
    <div className="grid gap-3">
      <Label>Access</Label>
      {presets(catalog).map((preset) => {
        const id = `api-token-preset-${preset.id}`;

        return (
          <div key={preset.id} className="flex items-start gap-2">
            <Checkbox
              id={id}
              checked={selectionState(preset.capabilities, value)}
              onCheckedChange={(checked) =>
                toggle(preset.capabilities, checked === true)
              }
              disabled={disabled}
            />
            <div className="grid gap-1">
              <Label htmlFor={id}>{preset.label}</Label>
              <p className="text-xs text-muted-foreground">
                {preset.description}
              </p>
            </div>
          </div>
        );
      })}
      <div className="grid max-h-96 divide-y overflow-y-auto rounded-md border">
        {catalog.groups.map((group) => {
          const names = group.capabilities.map((c) => c.name);
          const selectedCount = names.filter((n) => value.includes(n)).length;
          const groupId = `api-token-group-${group.label}`;

          return (
            <Collapsible key={group.label} className="group/section">
              <div className="flex items-center gap-2 px-3 py-2">
                <Checkbox
                  id={groupId}
                  checked={selectionState(names, value)}
                  onCheckedChange={(checked) =>
                    // a partly selected section fills in on the first click
                    toggle(names, checked !== false)
                  }
                  disabled={disabled}
                />
                <CollapsibleTrigger className="flex flex-1 items-center justify-between gap-2 text-left">
                  <span className="text-sm font-semibold">{group.label}</span>
                  <span className="flex items-center gap-1 text-xs text-muted-foreground">
                    {selectedCount} of {names.length}
                    <ChevronRightIcon className="h-4 w-4 transition-transform group-data-[state=open]/section:rotate-90" />
                  </span>
                </CollapsibleTrigger>
              </div>
              <CollapsibleContent className="grid gap-2 px-3 pb-3 pl-9">
                {group.capabilities.map((capability) => {
                  const id = `api-token-capability-${capability.name}`;

                  return (
                    <div
                      key={capability.name}
                      className="flex items-start gap-2"
                    >
                      <Checkbox
                        id={id}
                        className="mt-0.5 h-3 w-3 [&_svg]:h-3 [&_svg]:w-3"
                        checked={value.includes(capability.name)}
                        onCheckedChange={(checked) =>
                          toggle([capability.name], checked === true)
                        }
                        disabled={disabled}
                      />
                      <div className="grid gap-1">
                        <Label htmlFor={id} className="font-mono">
                          {capability.name}
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          {capability.description}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </CollapsibleContent>
            </Collapsible>
          );
        })}
      </div>
    </div>
  );
}

function ReviewCapabilities({
  catalog,
  values,
  isLoading,
  fieldErrors,
  onBack,
  onAccept,
}: {
  catalog: CapabilityCatalog;
  values: z.infer<typeof schema>;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  onBack: () => void;
  onAccept: () => void;
}) {
  const granted = grantedCapabilities(values.capabilities, catalog);
  const expiresIn = Object.entries(EXPIRES_IN_OPTS).find(
    ([, v]) => v === values.expiresIn,
  )?.[0];
  const errors = Object.values(fieldErrors ?? {});

  return (
    <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
      <DialogHeader>
        <DialogTitle>Review token access</DialogTitle>
      </DialogHeader>
      <p className="text-sm text-muted-foreground">
        <span className="font-medium text-foreground">{values.name}</span>
        {expiresIn && <> expires in {expiresIn}</>}. It will hold {granted.size}{' '}
        of {catalog.selectable.length} capabilities.
      </p>
      <div className="grid max-h-96 gap-4 overflow-y-auto">
        {catalog.groups.map((group) => (
          <div key={group.label} className="grid gap-2">
            <h3 className="text-sm font-semibold">{group.label}</h3>
            <div className="grid grid-cols-2 gap-x-6 gap-y-1 md:grid-cols-3">
              {group.capabilities.map((capability) => {
                const isGranted = granted.has(capability.name);

                return (
                  <div
                    key={capability.name}
                    className={cn(
                      'flex items-center gap-2 font-mono text-xs',
                      !isGranted && 'text-muted-foreground opacity-60',
                    )}
                  >
                    {isGranted ? (
                      <CheckIcon className="h-4 w-4 shrink-0 text-green-600 dark:text-green-500" />
                    ) : (
                      <Cross2Icon className="h-4 w-4 shrink-0" />
                    )}
                    {capability.name}
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
      {errors.length > 0 && (
        <div className="text-sm text-red-500">{errors.join(' ')}</div>
      )}
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onBack} disabled={isLoading}>
          Back
        </Button>
        <Button onClick={onAccept} disabled={isLoading}>
          {isLoading && <Spinner />}
          Accept and create
        </Button>
      </div>
    </DialogContent>
  );
}
