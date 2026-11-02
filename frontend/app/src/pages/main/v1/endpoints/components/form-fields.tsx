import { LabelRow } from '../lib/labels';
import { generateSigningSecret } from '../lib/platforms';
import { Copy, t } from './copy';
import { Button } from '@/components/v1/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/v1/ui/collapsible';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import EnvGroupArray, { KeyValueType } from '@/components/v1/ui/envvar';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Switch } from '@/components/v1/ui/switch';
import { ChevronRight, KeyRound } from 'lucide-react';
import { useState } from 'react';
import {
  FieldError,
  FieldErrors,
  UseFormRegister,
  UseFormRegisterReturn,
} from 'react-hook-form';

export const FieldErrorText = ({ error }: { error?: { message?: string } }) =>
  error?.message ? (
    <p className="text-xs text-red-500">{error.message}</p>
  ) : null;

export const RequiredMark = () => (
  <span className="text-red-500" aria-label={t('form.required')}>
    *
  </span>
);

export const OptionalMark = () => (
  <span className="text-muted-foreground">{t('form.optional')}</span>
);

export const SigningSecretField = ({
  inputProps,
  value,
  error,
  onGenerate,
  optional = false,
}: {
  inputProps: UseFormRegisterReturn;
  value: string;
  error?: FieldError;
  onGenerate: (secret: string) => void;
  // when editing, an empty secret keeps the current one
  optional?: boolean;
}) => (
  <div className="space-y-2">
    <Label htmlFor="signingSecret" className="text-sm font-medium">
      {t(
        optional
          ? 'form.signingSecret.rotateLabel'
          : 'form.signingSecret.label',
      )}{' '}
      {optional ? <OptionalMark /> : <RequiredMark />}
    </Label>
    <div className="flex flex-row items-center gap-2">
      <Input
        id="signingSecret"
        data-1p-ignore
        autoComplete="off"
        placeholder={t(
          optional
            ? 'form.signingSecret.rotatePlaceholder'
            : 'form.signingSecret.placeholder',
        )}
        className="h-10 font-mono text-xs"
        {...inputProps}
      />
      {value && <CopyToClipboard text={value} />}
      <Button
        type="button"
        variant="outline"
        className="shrink-0"
        onClick={() => onGenerate(generateSigningSecret())}
      >
        <KeyRound className="mr-2 size-4" />
        {t('form.signingSecret.generate')}
      </Button>
    </div>
    <FieldErrorText error={error} />
    <p className="text-xs text-muted-foreground">
      <Copy k="form.signingSecret.help" />
    </p>
  </div>
);

export type AdvancedFormFields = {
  requestTimeoutSeconds: number;
  pollIntervalSeconds: number;
  inlineWaitBudgetMs: number;
  labels: LabelRow[];
  enabled: boolean;
};

// EnvGroupArray edits KeyValueType rows; worker labels are never secret or locked
const toKeyValues = (rows: LabelRow[]): KeyValueType[] =>
  rows.map(({ key, value }) => ({
    key,
    value,
    hidden: false,
    locked: false,
    deleted: false,
    isEditing: true,
  }));

export const LabelsField = ({
  rows,
  onChange,
  error,
}: {
  rows: LabelRow[];
  onChange: (rows: LabelRow[]) => void;
  error?: { message?: string };
}) => (
  <div className="space-y-2">
    <Label>
      {t('form.labels.label')} <OptionalMark />
    </Label>
    <EnvGroupArray
      values={toKeyValues(rows)}
      setValues={(values) =>
        onChange(values.map(({ key, value }) => ({ key, value })))
      }
    />
    <FieldErrorText error={error} />
    <p className="text-xs text-muted-foreground">
      <Copy k="form.labels.help" />
    </p>
  </div>
);

export const AdvancedFields = ({
  register,
  errors,
  enabled,
  onEnabledChange,
  labels,
  onLabelsChange,
}: {
  register: UseFormRegister<AdvancedFormFields>;
  errors: FieldErrors<AdvancedFormFields>;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  labels: LabelRow[];
  onLabelsChange: (labels: LabelRow[]) => void;
}) => {
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className="flex flex-row items-center gap-1 text-sm font-medium"
        >
          <ChevronRight
            className={`size-4 transition-transform ${open ? 'rotate-90' : ''}`}
          />
          {t('form.advanced')}
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent className="mt-4 space-y-4 border-l-2 border-border pl-4">
        <div className="grid gap-4 md:grid-cols-3">
          <div className="space-y-2">
            <Label htmlFor="requestTimeoutSeconds">
              {t('form.requestTimeout.label')}
            </Label>
            <Input
              id="requestTimeoutSeconds"
              type="number"
              {...register('requestTimeoutSeconds', { valueAsNumber: true })}
            />
            <FieldErrorText error={errors.requestTimeoutSeconds} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="pollIntervalSeconds">
              {t('form.pollInterval.label')}
            </Label>
            <Input
              id="pollIntervalSeconds"
              type="number"
              {...register('pollIntervalSeconds', { valueAsNumber: true })}
            />
            <FieldErrorText error={errors.pollIntervalSeconds} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="inlineWaitBudgetMs">
              {t('form.inlineWait.label')}
            </Label>
            <Input
              id="inlineWaitBudgetMs"
              type="number"
              {...register('inlineWaitBudgetMs', { valueAsNumber: true })}
            />
            <FieldErrorText error={errors.inlineWaitBudgetMs} />
          </div>
        </div>
        <p className="text-xs text-muted-foreground">
          <Copy k="form.advanced.help" />
        </p>
        <LabelsField
          rows={labels}
          onChange={onLabelsChange}
          error={errors.labels?.root ?? errors.labels}
        />
        <div className="flex flex-row items-center justify-between rounded-lg border border-border p-4">
          <div className="space-y-0.5">
            <Label htmlFor="enabled" className="text-sm font-medium">
              {t('form.enabled.label')}
            </Label>
            <p className="text-xs text-muted-foreground">
              <Copy k="form.enabled.help" />
            </p>
          </div>
          <Switch
            id="enabled"
            checked={enabled}
            onCheckedChange={onEnabledChange}
          />
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
};
