import { cn } from '@/lib/utils';
import { Button } from '@/next/components/ui/button';
import { Label } from '@/next/components/ui/label';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import { TenantAlertingSettings } from '@/lib/api';
import { useState, forwardRef, useEffect } from 'react';
import { Switch } from '@/next/components/ui/switch';

const schema = z
  .object({
    maxAlertingFrequency: z.string().optional(),
    enableWorkflowRunFailureAlerts: z.boolean().optional(),
    enableExpiringTokenAlerts: z.boolean().optional(),
    enableTenantResourceLimitAlerts: z.boolean().optional(),
  })
  .refine(
    (data) => {
      // Only require maxAlertingFrequency if enableWorkflowRunFailureAlerts is true
      if (data.enableWorkflowRunFailureAlerts) {
        return !!data.maxAlertingFrequency;
      }
      return true;
    },
    {
      message:
        'Max alerting frequency is required when workflow run failure alerts are enabled',
      path: ['maxAlertingFrequency'],
    },
  );

interface UpdateTenantAlertingSettingsProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  alertingSettings: TenantAlertingSettings;
}

const SelectWithRef = forwardRef<HTMLButtonElement, any>((props, ref) => (
  <Select {...props}>
    <SelectTrigger className="w-[180px]" ref={ref}>
      <SelectValue id="maxAlertingFrequency" placeholder="Frequency..." />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="5m">5 minutes</SelectItem>
      <SelectItem value="1h">1 hour</SelectItem>
      <SelectItem value="24h">24 hours</SelectItem>
    </SelectContent>
  </Select>
));

SelectWithRef.displayName = 'SelectWithRef';

export function UpdateTenantAlertingSettings({
  className,
  ...props
}: UpdateTenantAlertingSettingsProps) {
  const [enabledWorkflowAlerting, setEnabledWorkflowAlerting] = useState(
    props.alertingSettings.enableWorkflowRunFailureAlerts,
  );

  const [enabledExpiringTokenAlerting, setEnabledExpiringTokenAlerting] =
    useState(props.alertingSettings.enableExpiringTokenAlerts);

  const [enableTenantResourceLimitAlerts, setEnableTenantResourceLimitAlerts] =
    useState(props.alertingSettings.enableTenantResourceLimitAlerts);

  const [hasChanges, setHasChanges] = useState(false);

  const {
    handleSubmit,
    control,
    formState: { errors },
    watch,
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      maxAlertingFrequency: props.alertingSettings.maxAlertingFrequency,
    },
  });

  const currentFrequency = watch('maxAlertingFrequency');

  useEffect(() => {
    const hasWorkflowChanges =
      enabledWorkflowAlerting !==
      props.alertingSettings.enableWorkflowRunFailureAlerts;
    const hasTokenChanges =
      enabledExpiringTokenAlerting !==
      props.alertingSettings.enableExpiringTokenAlerts;
    const hasResourceChanges =
      enableTenantResourceLimitAlerts !==
      props.alertingSettings.enableTenantResourceLimitAlerts;
    const hasFrequencyChanges =
      currentFrequency !== props.alertingSettings.maxAlertingFrequency;

    setHasChanges(
      hasWorkflowChanges ||
        hasTokenChanges ||
        hasResourceChanges ||
        hasFrequencyChanges,
    );
  }, [
    enabledWorkflowAlerting,
    enabledExpiringTokenAlerting,
    enableTenantResourceLimitAlerts,
    currentFrequency,
    props.alertingSettings,
  ]);

  const freqError =
    errors.maxAlertingFrequency?.message?.toString() || props.fieldErrors?.role;

  return (
    <div>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit({
            ...d,
            enableWorkflowRunFailureAlerts: enabledWorkflowAlerting,
            enableExpiringTokenAlerts: enabledExpiringTokenAlerting,
            enableTenantResourceLimitAlerts: enableTenantResourceLimitAlerts,
          });
        })}
        className={cn('grid gap-6', className)}
      >
        <div className="flex items-center space-x-2">
          <Switch
            id="eta"
            checked={enabledExpiringTokenAlerting}
            onClick={() => {
              setEnabledExpiringTokenAlerting((checkedState) => !checkedState);
            }}
          />
          <Label htmlFor="eta" className="text-sm">
            Enable Expiring Token Alerts
          </Label>
        </div>
        <div className="flex items-center space-x-2">
          <Switch
            id="atrl"
            checked={enableTenantResourceLimitAlerts}
            onClick={() => {
              setEnableTenantResourceLimitAlerts(
                (checkedState) => !checkedState,
              );
            }}
          />
          <Label htmlFor="atrl" className="text-sm">
            Enable Tenant Resource Limit Alerts
          </Label>
        </div>
        <div className="flex items-center space-x-2">
          <Switch
            id="awrf"
            checked={enabledWorkflowAlerting}
            onClick={() => {
              setEnabledWorkflowAlerting((checkedState) => !checkedState);
            }}
          />
          <Label htmlFor="awrf" className="text-sm">
            Enable Run Failure Alerts
          </Label>
        </div>

        <div className="flex flex-col gap-4">
          {enabledWorkflowAlerting && (
            <div className="grid gap-2">
              <Label htmlFor="maxAlertingFrequency">
                Max Run Failure Alerting Frequency
              </Label>
              <Controller
                control={control}
                name="maxAlertingFrequency"
                render={({ field }) => {
                  return (
                    <SelectWithRef onValueChange={field.onChange} {...field} />
                  );
                }}
              />
              {freqError && (
                <div className="text-sm text-red-500">{freqError}</div>
              )}
            </div>
          )}
          {hasChanges && (
            <Button loading={props.isLoading}>Save Changes</Button>
          )}
        </div>
      </form>
    </div>
  );
}
