import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Spinner } from '@/components/ui/loading.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { TenantAlertingSettings } from '@/lib/api';

const schema = z.object({
  maxAlertingFrequency: z.string(),
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
    errors.maxAlertingFrequency?.message?.toString() || props.fieldErrors?.role;

  return (
    <div className={cn('grid gap-6', className)}>
      <form
        onSubmit={handleSubmit((d) => {
          props.onSubmit(d);
        })}
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="maxAlertingFrequency">Max Alerting Frequency</Label>
            <Controller
              control={control}
              name="maxAlertingFrequency"
              render={({ field }) => {
                return (
                  <Select onValueChange={field.onChange} {...field}>
                    <SelectTrigger className="w-[180px]">
                      <SelectValue
                        id="maxAlertingFrequency"
                        placeholder="Frequency..."
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="5m">5 minutes</SelectItem>
                      <SelectItem value="1h">1 hour</SelectItem>
                      <SelectItem value="24h">24 hours</SelectItem>
                    </SelectContent>
                  </Select>
                );
              }}
            />
            {freqError && (
              <div className="text-sm text-red-500">{freqError}</div>
            )}
          </div>
          <Button disabled={props.isLoading}>
            {props.isLoading && <Spinner />}
            Update
          </Button>
        </div>
      </form>
    </div>
  );
}
