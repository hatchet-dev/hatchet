import { LabelRow } from '../../endpoints/lib/labels';
import { KINDS } from '../lib/kinds';
import { Button } from '@/components/v1/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/v1/ui/collapsible';
import EnvGroupArray, { KeyValueType } from '@/components/v1/ui/envvar';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Switch } from '@/components/v1/ui/switch';
import { Textarea } from '@/components/v1/ui/textarea';
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
        {optional ? 'Rotate API key' : 'API key'}{' '}
        {optional ? <OptionalMark /> : <RequiredMark />}
      </Label>
      <div className="flex flex-row items-center gap-2">
        <Input
          id="apiKey"
          type={visible ? 'text' : 'password'}
          data-1p-ignore
          autoComplete="off"
          placeholder={optional ? 'Leave blank to keep the current key' : ''}
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
        {KINDS[kind].keyHint} It is stored encrypted and never shown again.
      </p>
    </div>
  );
};

export type SharedFormFields = {
  defaultTemplate: string;
  maxIdleTtlSeconds: number;
  configJson: string;
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

export const ProviderFields = ({
  kind,
  register,
  errors,
  enabled,
  onEnabledChange,
  labels,
  onLabelsChange,
}: {
  kind: SandboxProviderKind;
  register: UseFormRegister<SharedFormFields>;
  errors: FieldErrors<SharedFormFields>;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  labels: LabelRow[];
  onLabelsChange: (labels: LabelRow[]) => void;
}) => {
  const [open, setOpen] = useState(false);

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
            Used when the SDK creates a sandbox without a template.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="maxIdleTtlSeconds">Max idle time (seconds)</Label>
          <Input
            id="maxIdleTtlSeconds"
            type="number"
            className="h-10"
            {...register('maxIdleTtlSeconds', { valueAsNumber: true })}
          />
          <FieldErrorText error={errors.maxIdleTtlSeconds} />
          <p className="text-xs text-muted-foreground">
            The longest a sandbox may sit idle before the vendor removes it.
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
          <div className="space-y-2">
            <Label htmlFor="configJson">
              Provider config <OptionalMark />
            </Label>
            <Textarea
              id="configJson"
              rows={5}
              placeholder={'{\n  "snapshotEveryTurn": false\n}'}
              className="font-mono text-xs"
              {...register('configJson')}
            />
            <FieldErrorText error={errors.configJson} />
            <p className="text-xs text-muted-foreground">
              Vendor settings as JSON. For Daytona: apiUrl, template, workdir
              and snapshotEveryTurn.
            </p>
          </div>
          <div className="space-y-2">
            <Label>
              Labels <OptionalMark />
            </Label>
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
                A disabled provider keeps its sandboxes but serves no turns.
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
