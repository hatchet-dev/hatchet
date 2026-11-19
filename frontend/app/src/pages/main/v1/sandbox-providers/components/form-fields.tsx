import { LabelRow } from '../../endpoints/lib/labels';
import { ConfigValues } from '../hooks/use-sandbox-providers';
import { ConfigField, KINDS } from '../lib/kinds';
import { Button } from '@/components/v1/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/v1/ui/collapsible';
import EnvGroupArray, { KeyValueType } from '@/components/v1/ui/envvar';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Switch } from '@/components/v1/ui/switch';
import { SandboxProviderKind } from '@/lib/api/generated/cloud/data-contracts';
import { ChevronRight, Eye, EyeOff } from 'lucide-react';
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
  <span className="text-red-500" aria-label="required">
    *
  </span>
);

export const OptionalMark = () => (
  <span className="text-muted-foreground">(optional)</span>
);

export const ApiKeyField = ({
  kind,
  inputProps,
  error,
  optional = false,
}: {
  kind: SandboxProviderKind;
  inputProps: UseFormRegisterReturn;
  error?: FieldError;
  optional?: boolean;
}) => {
  const [visible, setVisible] = useState(false);

  return (
    <div className="space-y-2">
      <Label htmlFor="apiKey" className="text-sm font-medium">
        {optional ? 'New API key' : 'API key'}{' '}
        {optional ? <OptionalMark /> : <RequiredMark />}
      </Label>
      <div className="flex flex-row items-center gap-2">
        <Input
          id="apiKey"
          type={visible ? 'text' : 'password'}
          data-1p-ignore
          autoComplete="off"
          placeholder={optional ? 'Leave empty to keep the current key' : ''}
          className="h-10 font-mono text-xs"
          {...inputProps}
        />
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="shrink-0"
          aria-label={visible ? 'Hide API key' : 'Show API key'}
          onClick={() => setVisible((v) => !v)}
        >
          {visible ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
        </Button>
      </div>
      <FieldErrorText error={error} />
      <p className="text-xs text-muted-foreground">
        {KINDS[kind].keyHint} Stored encrypted and never shown again.
      </p>
    </div>
  );
};

export type SharedFormFields = {
  defaultTemplate: string;
  maxIdleTtlSeconds: number;
  config: ConfigValues;
  labels: LabelRow[];
  enabled: boolean;
};

const toKeyValues = (rows: LabelRow[]): KeyValueType[] =>
  rows.map(({ key, value }) => ({
    key,
    value,
    hidden: false,
    locked: false,
    deleted: false,
    isEditing: true,
  }));

const ConfigInput = ({
  field,
  value,
  error,
  register,
  onChange,
}: {
  field: ConfigField;
  value: string | boolean | undefined;
  error?: { message?: string };
  register: UseFormRegister<SharedFormFields>;
  onChange: (value: string | boolean) => void;
}) => {
  const id = `config-${field.key}`;

  if (field.type === 'boolean') {
    return (
      <div className="flex flex-row items-center justify-between rounded-lg border border-border p-4">
        <div className="space-y-0.5">
          <Label htmlFor={id} className="text-sm font-medium">
            {field.label}
          </Label>
          <p className="text-xs text-muted-foreground">{field.help}</p>
        </div>
        <Switch
          id={id}
          checked={value === true}
          onCheckedChange={(checked) => onChange(checked)}
        />
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Label htmlFor={id}>
        {field.label} <OptionalMark />
      </Label>
      {field.type === 'select' ? (
        <Select
          value={typeof value === 'string' ? value : ''}
          onValueChange={onChange}
        >
          <SelectTrigger id={id} className="h-10">
            <SelectValue placeholder={field.options[0]} />
          </SelectTrigger>
          <SelectContent>
            {field.options.map((o) => (
              <SelectItem key={o} value={o}>
                {o}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <Input
          id={id}
          type={field.type === 'number' ? 'number' : 'text'}
          placeholder={field.placeholder}
          className={`h-10 ${field.type === 'text' && field.mono ? 'font-mono text-sm' : ''}`}
          {...register(`config.${field.key}`)}
        />
      )}
      <FieldErrorText error={error} />
      <p className="text-xs text-muted-foreground">{field.help}</p>
    </div>
  );
};

export const ProviderFields = ({
  kind,
  register,
  errors,
  enabled,
  onEnabledChange,
  config,
  onConfigChange,
  labels,
  onLabelsChange,
}: {
  kind: SandboxProviderKind;
  register: UseFormRegister<SharedFormFields>;
  errors: FieldErrors<SharedFormFields>;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  config: ConfigValues;
  onConfigChange: (key: string, value: string | boolean) => void;
  labels: LabelRow[];
  onLabelsChange: (labels: LabelRow[]) => void;
}) => {
  const [open, setOpen] = useState(false);
  const configErrors = errors.config as
    Record<string, { message?: string } | undefined> | undefined;

  return (
    <>
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="defaultTemplate">
            Default template <OptionalMark />
          </Label>
          <Input
            id="defaultTemplate"
            placeholder={KINDS[kind].templatePlaceholder}
            className="h-10 font-mono text-sm"
            {...register('defaultTemplate')}
          />
          <FieldErrorText error={errors.defaultTemplate} />
          <p className="text-xs text-muted-foreground">
            Used when the SDK does not name one.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="maxIdleTtlSeconds">Max idle seconds</Label>
          <Input
            id="maxIdleTtlSeconds"
            type="number"
            className="h-10"
            {...register('maxIdleTtlSeconds', { valueAsNumber: true })}
          />
          <FieldErrorText error={errors.maxIdleTtlSeconds} />
          <p className="text-xs text-muted-foreground">
            Idle sandboxes are removed after this long.
          </p>
        </div>
      </div>

      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger asChild>
          <button
            type="button"
            className="flex flex-row items-center gap-1 text-sm font-medium"
          >
            <ChevronRight
              className={`size-4 transition-transform ${open ? 'rotate-90' : ''}`}
            />
            Advanced
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent className="mt-4 space-y-4 border-l-2 border-border pl-4">
          {KINDS[kind].fields.map((field) => (
            <ConfigInput
              key={field.key}
              field={field}
              value={config[field.key]}
              error={configErrors?.[field.key]}
              register={register}
              onChange={(value) => onConfigChange(field.key, value)}
            />
          ))}
          <div className="space-y-2">
            <Label>
              Metadata <OptionalMark />
            </Label>
            <p className="text-xs text-muted-foreground">
              Default metadata for every sandbox this provider creates. The SDK
              can override it key by key. It shows on the sandbox, its task runs
              and at the vendor.
            </p>
            <EnvGroupArray
              values={toKeyValues(labels)}
              setValues={(values) =>
                onLabelsChange(values.map(({ key, value }) => ({ key, value })))
              }
            />
            <FieldErrorText error={errors.labels?.root ?? errors.labels} />
          </div>
          <div className="flex flex-row items-center justify-between rounded-lg border border-border p-4">
            <div className="space-y-0.5">
              <Label htmlFor="enabled" className="text-sm font-medium">
                Enabled
              </Label>
              <p className="text-xs text-muted-foreground">
                Disabled providers keep their sandboxes but serve no turns.
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
    </>
  );
};
