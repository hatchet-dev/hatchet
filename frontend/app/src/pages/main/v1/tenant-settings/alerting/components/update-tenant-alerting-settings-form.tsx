import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Switch } from '@/components/v1/ui/switch';
import { TenantAlertingSettings } from '@/lib/api';
import { cn } from '@/lib/utils';
import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

const schema = z.object({
  maxAlertingFrequency: z.string(),
  enableWorkflowRunFailureAlerts: z.boolean().optional(),
  enableExpiringTokenAlerts: z.boolean().optional(),
  enableTenantResourceLimitAlerts: z.boolean().optional(),
});

interface UpdateTenantAlertingSettingsProps {
  className?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
  alertingSettings: TenantAlertingSettings;
}

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

  const {
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      maxAlertingFrequency: props.alertingSettings.maxAlertingFrequency,
    },
  });

  const freqError =
    errors.maxAlertingFrequency?.message?.toString() ||
    props.fieldErrors?.maxAlertingFrequency;

  return (
    <form
      onSubmit={handleSubmit((d) => {
        props.onSubmit({
          ...d,
          enableWorkflowRunFailureAlerts: enabledWorkflowAlerting,
          enableExpiringTokenAlerts: enabledExpiringTokenAlerting,
          enableTenantResourceLimitAlerts: enableTenantResourceLimitAlerts,
        });
      })}
      className={cn('divide-y divide-border', className)}
    >
      <AlertSettingRow
        label="Expiring Token Alerts"
        description="Send alerts when API tokens are approaching their expiration date"
      >
        <Switch
          id="eta"
          checked={enabledExpiringTokenAlerting}
          onClick={() => setEnabledExpiringTokenAlerting((s) => !s)}
        />
      </AlertSettingRow>

      <AlertSettingRow
        label="Resource Limit Alerts"
        description="Send alerts when this tenant is approaching (or exceeds) resource limits"
      >
        <Switch
          id="atrl"
          checked={enableTenantResourceLimitAlerts}
          onClick={() => setEnableTenantResourceLimitAlerts((s) => !s)}
        />
      </AlertSettingRow>

      <AlertSettingRow
        label="Run Failure Alerts"
        description="Send alerts when workflow runs fail"
      >
        <Switch
          id="awrf"
          checked={enabledWorkflowAlerting}
          onClick={() => setEnabledWorkflowAlerting((s) => !s)}
        />
      </AlertSettingRow>

      {enabledWorkflowAlerting && (
        <AlertSettingRow
          label="Max Failure Alert Frequency"
          description="Limit how often repeated run-failure alerts are sent for the same ongoing issue."
        >
          <div className="flex flex-col items-end gap-1">
            <Controller
              control={control}
              name="maxAlertingFrequency"
              render={({ field }) => (
                <Select onValueChange={field.onChange} {...field}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue placeholder="Frequency..." />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="5m">5 minutes</SelectItem>
                    <SelectItem value="1h">1 hour</SelectItem>
                    <SelectItem value="24h">24 hours</SelectItem>
                  </SelectContent>
                </Select>
              )}
            />
            {freqError && (
              <div className="text-sm text-red-500">{freqError}</div>
            )}
          </div>
        </AlertSettingRow>
      )}

      <div className="flex justify-end pt-4">
        <Button disabled={props.isLoading} className="w-fit">
          {props.isLoading && <Spinner />}
          Save
        </Button>
      </div>
    </form>
  );
}

function AlertSettingRow({
  label,
  description,
  children,
}: {
  label: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-6 py-4">
      <div className="max-w-2xl">
        <p className="text-sm font-medium">{label}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
      </div>
      {children}
    </div>
  );
}
