import {
  buildCreateRequest,
  EndpointCreateFormData,
  endpointCreateFormSchema,
  resolveCreateUrls,
  useEndpointMutations,
  usesDerivedUrls,
} from '../hooks/use-endpoints';
import { PLATFORM_ORDER, PLATFORMS } from '../lib/platforms';
import { Copy, t } from './copy';
import {
  AdvancedFields,
  AdvancedFormFields,
  FieldErrorText,
  RequiredMark,
  SigningSecretField,
} from './form-fields';
import { PlatformIcon } from './platform-name';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
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
import { Switch } from '@/components/v1/ui/switch';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { V1ServerlessEndpointKind } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError, isAxiosError } from 'axios';
import { ArrowLeft, BookOpen, Cloud, Code2, ShieldAlert } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useForm, UseFormRegister } from 'react-hook-form';

const defaultValues = (
  kind: V1ServerlessEndpointKind,
): EndpointCreateFormData => ({
  kind,
  name: '',
  baseUrl: '',
  manualUrls: false,
  healthcheckUrl: '',
  triggerUrl: '',
  signingSecret: '',
  requestTimeoutSeconds: PLATFORMS[kind].defaultRequestTimeoutSeconds,
  pollIntervalSeconds: 30,
  inlineWaitBudgetMs: 5000,
  labels: [],
  enabled: true,
});

const NOT_ENTITLED = 'the serverless operator is not enabled for this tenant';

export const CreateEndpointDialog = ({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) => {
  const { tenantId } = useCurrentTenantId();
  const { isSelfHosted } = useControlPlane();
  const navigate = useNavigate();
  const { create } = useEndpointMutations();
  const [kind, setKind] = useState<V1ServerlessEndpointKind | undefined>();
  const [apiErrors, setApiErrors] = useState<string[]>([]);
  const [notEntitled, setNotEntitled] = useState(false);
  const { handleApiError } = useApiError({ setErrors: setApiErrors });

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<EndpointCreateFormData>({
    resolver: zodResolver(endpointCreateFormSchema),
    defaultValues: defaultValues(V1ServerlessEndpointKind.CLOUDFLARE_WORKERS),
  });

  const values = watch();
  const derived = usesDerivedUrls(values);
  const resolvedUrls = resolveCreateUrls(values);

  const close = useCallback(() => {
    onOpenChange(false);
    setKind(undefined);
    setApiErrors([]);
    setNotEntitled(false);
  }, [onOpenChange]);

  const pickPlatform = (picked: V1ServerlessEndpointKind) => {
    reset(defaultValues(picked));
    setKind(picked);
    setApiErrors([]);
    setNotEntitled(false);
  };

  const onSubmit = (data: EndpointCreateFormData) => {
    setApiErrors([]);
    setNotEntitled(false);

    create.mutate(buildCreateRequest(data), {
      onSuccess: (endpoint) => {
        close();
        navigate({
          to: appRoutes.tenantEndpointRoute.to,
          params: { tenant: tenantId, endpoint: endpoint.metadata.id },
        });
      },
      onError: (error) => {
        if (
          isAxiosError(error) &&
          error.response?.status === 403 &&
          JSON.stringify(error.response.data).includes(NOT_ENTITLED)
        ) {
          setNotEntitled(true);
          return;
        }

        handleApiError(error as AxiosError);
      },
    });
  };

  const platform = kind ? PLATFORMS[kind] : undefined;

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => (isOpen ? onOpenChange(true) : close())}
    >
      <DialogContent className="max-h-[90dvh] max-w-[90%] overflow-y-auto md:max-w-[80%] lg:max-w-[60%] xl:max-w-[50%]">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center gap-x-3">
            <div className="flex size-8 items-center justify-center rounded-full bg-blue-100">
              {kind ? (
                <PlatformIcon kind={kind} className="text-indigo-700" />
              ) : (
                <Cloud className="size-4 text-indigo-700" />
              )}
            </div>
            {kind
              ? t('create.titleForPlatform', {
                  platform: { key: `platform.${kind}.label` },
                })
              : t('create.title')}
          </DialogTitle>
        </DialogHeader>

        {!kind || !platform ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              <Copy k="create.pickPlatform" />
            </p>
            <div className="grid gap-3 md:grid-cols-3">
              {PLATFORM_ORDER.map((k) => (
                <button
                  key={k}
                  type="button"
                  onClick={() => pickPlatform(k)}
                  className="flex flex-col items-start gap-2 rounded-lg border border-border p-4 text-left transition-colors hover:border-primary hover:bg-muted/50"
                >
                  <PlatformIcon kind={k} className="size-6" />
                  <span className="text-sm font-medium">
                    {t(`platform.${k}.label`)}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    <Copy k={`platform.${k}.description`} />
                  </span>
                </button>
              ))}
            </div>
            <div className="flex flex-row items-start gap-2 rounded-md bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
              <Code2 className="mt-0.5 size-3.5 shrink-0" />
              <span>
                <Copy k="create.programmatic" />
              </span>
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
              {t('create.back')}
            </button>

            <div className="flex flex-row items-start gap-3 rounded-lg border border-border p-4 text-sm">
              <BookOpen className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
              <p>
                <Copy k={`platform.${kind}.guide`} />
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="name" className="text-sm font-medium">
                {t('form.name.label')} <RequiredMark />
              </Label>
              <Input
                id="name"
                data-1p-ignore
                placeholder={t('form.name.placeholder')}
                className="h-10"
                {...register('name')}
              />
              <FieldErrorText error={errors.name} />
            </div>

            {platform.healthcheckPath && (
              <div className="flex flex-row items-center justify-between">
                <Label htmlFor="manualUrls" className="text-xs">
                  {t('form.manualUrls')}
                </Label>
                <Switch
                  id="manualUrls"
                  checked={values.manualUrls}
                  onCheckedChange={(checked) => {
                    setValue('manualUrls', checked);

                    if (checked) {
                      setValue('healthcheckUrl', resolvedUrls.healthcheckUrl);
                      setValue('triggerUrl', resolvedUrls.triggerUrl);
                    }
                  }}
                />
              </div>
            )}

            {derived && kind !== V1ServerlessEndpointKind.GENERIC_HTTP ? (
              <div className="space-y-2">
                <Label htmlFor="baseUrl" className="text-sm font-medium">
                  {t(`platform.${kind}.baseUrlLabel`)} <RequiredMark />
                </Label>
                <Input
                  id="baseUrl"
                  placeholder={platform.baseUrlPlaceholder}
                  className="h-10 font-mono text-sm"
                  {...register('baseUrl')}
                />
                <FieldErrorText error={errors.baseUrl} />
                <div className="space-y-1 rounded-md bg-muted px-3 py-2 font-mono text-xs text-muted-foreground">
                  <div>
                    <span className="text-foreground">healthcheck</span>{' '}
                    {resolvedUrls.healthcheckUrl ||
                      `…${platform.healthcheckPath}`}
                  </div>
                  <div>
                    <span className="text-foreground">trigger</span>{' '}
                    {resolvedUrls.triggerUrl || `…${platform.triggerPath}`}
                  </div>
                </div>
              </div>
            ) : (
              <>
                <div className="space-y-2">
                  <Label
                    htmlFor="healthcheckUrl"
                    className="text-sm font-medium"
                  >
                    {t('form.healthcheckUrl.label')} <RequiredMark />
                  </Label>
                  <Input
                    id="healthcheckUrl"
                    placeholder="https://api.example.com/hatchet/healthcheck"
                    className="h-10 font-mono text-sm"
                    {...register('healthcheckUrl')}
                  />
                  <FieldErrorText error={errors.healthcheckUrl} />
                  <p className="text-xs text-muted-foreground">
                    <Copy k="form.healthcheckUrl.help" />
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="triggerUrl" className="text-sm font-medium">
                    {t('form.triggerUrl.label')} <RequiredMark />
                  </Label>
                  <Input
                    id="triggerUrl"
                    placeholder="https://api.example.com/hatchet/trigger"
                    className="h-10 font-mono text-sm"
                    {...register('triggerUrl')}
                  />
                  <FieldErrorText error={errors.triggerUrl} />
                  <p className="text-xs text-muted-foreground">
                    <Copy k="form.triggerUrl.help" />
                  </p>
                </div>
              </>
            )}

            <SigningSecretField
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
                setValue('labels', labels, {
                  shouldValidate: !!errors.labels,
                })
              }
            />

            {notEntitled && (
              <Alert variant="destructive">
                <ShieldAlert className="size-4" />
                <AlertTitle>{t('create.notEntitled.title')}</AlertTitle>
                <AlertDescription className="space-y-2 text-xs">
                  {isSelfHosted ? (
                    <p>
                      <Copy k="create.notEntitled.oss" />
                    </p>
                  ) : (
                    <>
                      <p>
                        <Copy k="create.notEntitled.cloud" />
                      </p>
                      <Button asChild size="sm" variant="outline">
                        <a href={t('create.notEntitled.supportUrl')}>
                          {t('create.notEntitled.contactSupport')}
                        </a>
                      </Button>
                    </>
                  )}
                </AlertDescription>
              </Alert>
            )}

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
                {t('form.cancel')}
              </Button>
              <Button
                type="submit"
                disabled={create.isPending}
                className={cn(create.isPending && 'gap-2')}
              >
                {create.isPending && <Spinner />}
                {t('create.submit')}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
