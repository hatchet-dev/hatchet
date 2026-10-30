import {
  buildCreateRequest,
  defaultCreateValues,
  ProviderCreateFormData,
  providerCreateFormSchema,
  useSandboxProviderMutations,
} from '../hooks/use-sandbox-providers';
import { KIND_ORDER, KINDS } from '../lib/kinds';
import {
  ApiKeyField,
  FieldErrorText,
  ProviderFields,
  RequiredMark,
  SharedFormFields,
} from './form-fields';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { ArrowLeft, Box } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useForm, UseFormRegister } from 'react-hook-form';

export const CreateProviderDialog = ({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { tenantId } = useCurrentTenantId();
  const navigate = useNavigate();
  const { create } = useSandboxProviderMutations();
  const [kind, setKind] = useState<SandboxProviderKind | undefined>();
  const [apiErrors, setApiErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setApiErrors });

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<ProviderCreateFormData>({
    resolver: zodResolver(providerCreateFormSchema),
    defaultValues: defaultCreateValues(SandboxProviderKind.DAYTONA),
  });

  const values = watch();

  const close = useCallback(() => {
    onOpenChange(false);
    setKind(undefined);
    setApiErrors([]);
  }, [onOpenChange]);

  const pickKind = (picked: SandboxProviderKind) => {
    reset(defaultCreateValues(picked));
    setKind(picked);
    setApiErrors([]);
  };

  const onSubmit = (data: ProviderCreateFormData) => {
    setApiErrors([]);

    create.mutate(buildCreateRequest(data), {
      onSuccess: (provider) => {
        close();
        navigate({
          to: appRoutes.tenantSandboxProviderRoute.to,
          params: { tenant: tenantId, sandboxProvider: provider.metadata.id },
        });
      },
      onError: (error) => handleApiError(error as AxiosError),
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => (isOpen ? onOpenChange(true) : close())}
    >
      <DialogContent className="max-h-[90dvh] max-w-[90%] overflow-y-auto md:max-w-[80%] lg:max-w-[60%] xl:max-w-[50%]">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center gap-x-3">
            <div className="flex size-8 items-center justify-center rounded-full bg-blue-100">
              <Box className="size-4 text-indigo-700" />
            </div>
            {kind
              ? `Connect ${KINDS[kind].label}`
              : 'Connect a sandbox provider'}
          </DialogTitle>
        </DialogHeader>

        {!kind ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Pick the vendor that runs your sandboxes. Hatchet keeps the
              snapshot lineage and serves the sandbox SDK; the vendor runs the
              machines under your own account.
            </p>
            <div className="grid gap-3 md:grid-cols-2">
              {KIND_ORDER.map((k) => (
                <button
                  key={k}
                  type="button"
                  onClick={() => pickKind(k)}
                  className="flex flex-col items-start gap-2 rounded-lg border border-border p-4 text-left transition-colors hover:border-primary hover:bg-muted/50"
                >
                  <span className="text-sm font-medium">{KINDS[k].label}</span>
                  <span className="text-xs text-muted-foreground">
                    {KINDS[k].description}
                    {!KINDS[k].available && ' Driver not available yet.'}
                  </span>
                </button>
              ))}
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <button
              type="button"
              onClick={() => setKind(undefined)}
              className="flex flex-row items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            >
              <ArrowLeft className="size-3" />
              Choose another vendor
            </button>

            <div className="space-y-2">
              <Label htmlFor="name" className="text-sm font-medium">
                Name <RequiredMark />
              </Label>
              <Input
                id="name"
                data-1p-ignore
                placeholder={KINDS[kind].label.toLowerCase()}
                className="h-10"
                {...register('name')}
              />
              <FieldErrorText error={errors.name} />
              <p className="text-xs text-muted-foreground">
                The SDK picks this provider with metadata.provider when the
                tenant has more than one.
              </p>
            </div>

            <ApiKeyField
              kind={kind}
              inputProps={register('apiKey')}
              error={errors.apiKey}
            />

            <ProviderFields
              kind={kind}
              register={
                register as unknown as UseFormRegister<SharedFormFields>
              }
              errors={errors}
              enabled={values.enabled}
              onEnabledChange={(checked) => setValue('enabled', checked)}
              labels={values.labels}
              onLabelsChange={(labels) =>
                setValue('labels', labels, { shouldValidate: !!errors.labels })
              }
            />

            {apiErrors.length > 0 && (
              <Alert variant="destructive">
                <AlertDescription className="space-y-1 text-xs">
                  {apiErrors.map((e) => (
                    <div key={e}>{e}</div>
                  ))}
                </AlertDescription>
              </Alert>
            )}

            <div className="flex justify-end gap-3 pt-2">
              <Button type="button" variant="outline" onClick={close}>
                Cancel
              </Button>
              <Button type="submit" disabled={create.isPending}>
                {create.isPending && <Spinner />}
                Connect provider
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
