import {
  buildUpdateRequest,
  EndpointUpdateFormData,
  endpointUpdateFormSchema,
  useEndpointMutations,
} from '../hooks/use-endpoints';
import { labelsToRows } from '../lib/labels';
import { PLATFORM_ORDER, PLATFORMS } from '../lib/platforms';
import { Copy, t } from './copy';
import {
  AdvancedFields,
  AdvancedFormFields,
  FieldErrorText,
  SigningSecretField,
} from './form-fields';
import { PlatformName } from './platform-name';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { V1ServerlessEndpoint, V1ServerlessEndpointKind } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';
import { useEffect, useState } from 'react';
import { useForm, UseFormRegister } from 'react-hook-form';

const formValues = (
  endpoint: V1ServerlessEndpoint,
): EndpointUpdateFormData => ({
  kind: endpoint.kind,
  name: endpoint.name,
  healthcheckUrl: endpoint.healthcheckUrl,
  triggerUrl: endpoint.triggerUrl,
  signingSecret: '',
  requestTimeoutSeconds: endpoint.requestTimeoutSeconds,
  pollIntervalSeconds: endpoint.pollIntervalSeconds,
  inlineWaitBudgetMs: endpoint.inlineWaitBudgetMs,
  labels: labelsToRows(endpoint.labels),
  enabled: endpoint.enabled,
});

export const EditEndpointDialog = ({
  endpoint,
  open,
  onOpenChange,
}: {
  endpoint: V1ServerlessEndpoint;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { update } = useEndpointMutations();
  const [apiErrors, setApiErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setApiErrors });

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<EndpointUpdateFormData>({
    resolver: zodResolver(endpointUpdateFormSchema),
    defaultValues: formValues(endpoint),
  });

  // reset only when the dialog opens, so background refetches don't wipe edits
  useEffect(() => {
    if (open) {
      reset(formValues(endpoint));
      setApiErrors([]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const values = watch();

  const onSubmit = (data: EndpointUpdateFormData) => {
    setApiErrors([]);

    update.mutate(
      {
        endpointId: endpoint.metadata.id,
        data: buildUpdateRequest(data, endpoint.labels),
      },
      {
        onSuccess: () => onOpenChange(false),
        onError: (error) => handleApiError(error as AxiosError),
      },
    );
  };

  const platform = PLATFORMS[values.kind];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] max-w-[90%] overflow-y-auto md:max-w-[70%] lg:max-w-[50%]">
        <DialogHeader>
          <DialogTitle>{t('edit.title', { name: endpoint.name })}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="edit-name">{t('form.name.label')}</Label>
              <Input id="edit-name" className="h-10" {...register('name')} />
              <FieldErrorText error={errors.name} />
            </div>
            <div className="space-y-2">
              <Label>{t('edit.platform')}</Label>
              <Select
                value={values.kind}
                onValueChange={(value: V1ServerlessEndpointKind) =>
                  setValue('kind', value)
                }
              >
                <SelectTrigger className="h-10">
                  <SelectValue>
                    <PlatformName kind={values.kind} />
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {PLATFORM_ORDER.map((k) => (
                    <SelectItem key={k} value={k}>
                      <PlatformName kind={k} />
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-healthcheckUrl">
              {t('form.healthcheckUrl.label')}
            </Label>
            <Input
              id="edit-healthcheckUrl"
              className="h-10 font-mono text-sm"
              {...register('healthcheckUrl')}
            />
            <FieldErrorText error={errors.healthcheckUrl} />
            {platform.healthcheckPath && (
              <p className="text-xs text-muted-foreground">
                <Copy
                  k="edit.adapterPath"
                  params={{
                    platform: { key: `platform.${values.kind}.label` },
                    path: platform.healthcheckPath,
                  }}
                />
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-triggerUrl">
              {t('form.triggerUrl.label')}
            </Label>
            <Input
              id="edit-triggerUrl"
              className="h-10 font-mono text-sm"
              {...register('triggerUrl')}
            />
            <FieldErrorText error={errors.triggerUrl} />
            {platform.triggerPath && (
              <p className="text-xs text-muted-foreground">
                <Copy
                  k="edit.adapterPath"
                  params={{
                    platform: { key: `platform.${values.kind}.label` },
                    path: platform.triggerPath,
                  }}
                />
              </p>
            )}
          </div>

          <SigningSecretField
            optional
            inputProps={register('signingSecret')}
            value={values.signingSecret}
            error={errors.signingSecret}
            onGenerate={(secret) =>
              setValue('signingSecret', secret, { shouldValidate: true })
            }
          />

          <AdvancedFields
            register={
              register as unknown as UseFormRegister<AdvancedFormFields>
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
            <DialogClose asChild>
              <Button type="button" variant="outline">
                {t('form.cancel')}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={update.isPending}>
              {update.isPending && <Spinner />}
              {t('edit.submit')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
