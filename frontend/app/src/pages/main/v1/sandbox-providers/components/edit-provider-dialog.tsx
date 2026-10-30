import {
  buildUpdateRequest,
  ProviderUpdateFormData,
  providerUpdateFormSchema,
  updateFormValues,
  useSandboxProviderMutations,
} from '../hooks/use-sandbox-providers';
import { KINDS } from '../lib/kinds';
import {
  ApiKeyField,
  FieldErrorText,
  ProviderFields,
  SharedFormFields,
} from './form-fields';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';
import { useEffect, useState } from 'react';
import { useForm, UseFormRegister } from 'react-hook-form';

export const EditProviderDialog = ({
  provider,
  open,
  onOpenChange,
}: {
  provider: SandboxProvider;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { update } = useSandboxProviderMutations();
  const [apiErrors, setApiErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setApiErrors });

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<ProviderUpdateFormData>({
    resolver: zodResolver(providerUpdateFormSchema),
    defaultValues: updateFormValues(provider),
  });

  useEffect(() => {
    if (open) {
      reset(updateFormValues(provider));
      setApiErrors([]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const values = watch();

  const onSubmit = (data: ProviderUpdateFormData) => {
    setApiErrors([]);

    update.mutate(
      {
        providerId: provider.metadata.id,
        data: buildUpdateRequest(data, provider),
      },
      {
        onSuccess: () => onOpenChange(false),
        onError: (error) => handleApiError(error as AxiosError),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] max-w-[90%] overflow-y-auto md:max-w-[70%] lg:max-w-[50%]">
        <DialogHeader>
          <DialogTitle>Edit {provider.name}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input id="edit-name" className="h-10" {...register('name')} />
              <FieldErrorText error={errors.name} />
            </div>
            <div className="space-y-2">
              <Label>Vendor</Label>
              <Input
                className="h-10"
                value={KINDS[provider.kind].label}
                disabled
              />
            </div>
          </div>

          <ApiKeyField
            optional
            kind={provider.kind}
            inputProps={register('apiKey')}
            error={errors.apiKey}
          />

          <ProviderFields
            kind={provider.kind}
            register={register as unknown as UseFormRegister<SharedFormFields>}
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
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={update.isPending}>
              {update.isPending && <Spinner />}
              Save changes
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
