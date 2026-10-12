import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { cn } from '@/lib/utils';
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CopyIcon,
  Cross2Icon,
  PlusIcon,
  TrashIcon,
} from '@radix-ui/react-icons';
import {
  ADDITIONAL_PROPERTY_FLAG,
  ArrayFieldItemTemplateProps,
  buttonId,
  getTemplate,
  getUiOptions,
  IconButtonProps,
  RegistryWidgetsType,
  TemplatesType,
  WidgetProps,
  WrapIfAdditionalTemplateProps,
} from '@rjsf/utils';
import { format } from 'date-fns';
import { ComponentType } from 'react';

function AddButton({ onClick, disabled, className }: IconButtonProps) {
  return (
    <div className={className}>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={onClick}
        disabled={disabled}
      >
        <PlusIcon className="mr-1 size-4" />
        Add
      </Button>
    </div>
  );
}

function iconButton(
  Icon: ComponentType<{ className?: string }>,
  label: string,
  className?: string,
) {
  return function IconButton({ onClick, disabled }: IconButtonProps) {
    return (
      <Button
        type="button"
        variant="ghost"
        size="icon"
        title={label}
        aria-label={label}
        onClick={onClick}
        disabled={disabled}
        className={className}
      >
        <Icon className="size-4" />
      </Button>
    );
  };
}

function ArrayFieldItemTemplate({
  children,
  buttonsProps,
  hasToolbar,
  registry,
  uiSchema,
}: ArrayFieldItemTemplateProps) {
  const ArrayFieldItemButtonsTemplate = getTemplate(
    'ArrayFieldItemButtonsTemplate',
    registry,
    getUiOptions(uiSchema),
  );

  return (
    <div className="flex items-start gap-1">
      <div className="min-w-0 flex-1">{children}</div>
      {hasToolbar && (
        <div className="flex shrink-0">
          <ArrayFieldItemButtonsTemplate {...buttonsProps} />
        </div>
      )}
    </div>
  );
}

// Map entries (dict[str, X]) render as key input + value + remove. Plain values
// fit on one row; lists and objects get a card with the key as its header.
function WrapIfAdditionalTemplate({
  id,
  classNames,
  style,
  disabled,
  readonly,
  label,
  onKeyRenameBlur,
  onRemoveProperty,
  schema,
  hideError,
  rawErrors,
  children,
  uiSchema,
  registry,
}: WrapIfAdditionalTemplateProps) {
  const className = cn(
    'form-group',
    classNames,
    !hideError && rawErrors?.length && 'has-error has-danger',
  );

  if (!(ADDITIONAL_PROPERTY_FLAG in schema)) {
    return (
      <div className={className} style={style}>
        {children}
      </div>
    );
  }

  const { RemoveButton } = registry.templates.ButtonTemplates;
  const keyInput = (
    <Input
      // remount on rename so defaultValue follows the new key
      key={label}
      id={`${id}-key`}
      aria-label="Key"
      placeholder="Key"
      defaultValue={label}
      onBlur={onKeyRenameBlur}
      disabled={disabled || readonly}
    />
  );
  const removeButton = (
    <RemoveButton
      id={buttonId(id, 'remove')}
      disabled={disabled || readonly}
      onClick={onRemoveProperty}
      uiSchema={uiSchema}
      registry={registry}
    />
  );

  if (schema.type !== 'object' && schema.type !== 'array') {
    return (
      <div className={cn(className, 'flex items-start gap-1')} style={style}>
        <div className="w-1/3 shrink-0">{keyInput}</div>
        <div className="min-w-0 flex-1">{children}</div>
        {removeButton}
      </div>
    );
  }

  return (
    <div className={cn(className, 'rounded-md border p-2')} style={style}>
      <div className="mb-2 flex items-center gap-1">
        <div className="min-w-0 flex-1">{keyInput}</div>
        {removeButton}
      </div>
      <div className="border-l pl-3">{children}</div>
    </div>
  );
}

export const templates: Partial<TemplatesType> = {
  ArrayFieldItemTemplate,
  WrapIfAdditionalTemplate,
  ButtonTemplates: {
    AddButton,
    ClearButton: iconButton(Cross2Icon, 'Clear'),
    CopyButton: iconButton(CopyIcon, 'Duplicate'),
    MoveUpButton: iconButton(ArrowUpIcon, 'Move up'),
    MoveDownButton: iconButton(ArrowDownIcon, 'Move down'),
    RemoveButton: iconButton(TrashIcon, 'Remove', 'text-red-500'),
    // the trigger dialog has its own submit button
    SubmitButton: () => null,
  },
};

function parseDate(value: unknown, dateOnly: boolean): Date | undefined {
  if (typeof value !== 'string' || !value) {
    return undefined;
  }
  // a bare yyyy-MM-dd parses as UTC midnight; read it as local midnight
  const date = new Date(dateOnly ? `${value}T00:00:00` : value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

function dateWidget(dateOnly: boolean) {
  return function DateWidget({ value, onChange, schema }: WidgetProps) {
    return (
      <div className="flex items-center gap-1">
        <DateTimePicker
          date={parseDate(value, dateOnly)}
          setDate={(date) =>
            onChange(
              date
                ? dateOnly
                  ? format(date, 'yyyy-MM-dd')
                  : date.toISOString()
                : undefined,
            )
          }
          label={schema.title ?? 'Date'}
          triggerClassName="h-9 w-full text-sm"
        />
        {value && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            title="Clear"
            aria-label="Clear"
            onClick={() => onChange(undefined)}
          >
            <Cross2Icon className="size-4" />
          </Button>
        )}
      </div>
    );
  };
}

export const widgets: RegistryWidgetsType = {
  DateWidget: dateWidget(true),
  DateTimeWidget: dateWidget(false),
};
