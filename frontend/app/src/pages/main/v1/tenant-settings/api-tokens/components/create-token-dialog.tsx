import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Button } from '@/components/v1/ui/button';
import { z } from 'zod';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/v1/ui/label';
import { Input } from '@/components/v1/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { SecretCopier } from '@/components/v1/ui/secret-copier';

export const EXPIRES_IN_OPTS = {
  '3 months': `${3 * 30 * 24 * 60 * 60}s`,
  '1 year': `${365 * 24 * 60 * 60}s`,
  '100 years': `${100 * 365 * 24 * 60 * 60}s`,
};

const schema = z.object({
  name: z.string().min(1).max(255),
  expiresIn: z.string().optional(),
});

interface CreateTokenDialogProps {
  className?: string;
  token?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateTokenDialog({
  className,
  token,
  ...props
}: CreateTokenDialogProps) {
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;

  if (token) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          This is the only time we will show you this token. Make sure to copy
          it somewhere safe.
        </p>
        <SecretCopier
          secrets={{ HATCHET_CLIENT_TOKEN: token }}
          className="text-sm"
          maxWidth={'calc(700px - 4rem)'}
          copy
        />
      </DialogContent>
    );
  }

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create a new API token</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="email">Name</Label>
              <Input
                {...register('name')}
                id="api-token-name"
                placeholder="My Token"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
            </div>
            <Label htmlFor="expiresIn">Expires In</Label>
            <Controller
              control={control}
              defaultValue={EXPIRES_IN_OPTS['100 years']}
              name="expiresIn"
              render={({ field }) => {
                return (
                  <Select onValueChange={field.onChange} {...field}>
                    <SelectTrigger id="expiresIn">
                      <SelectValue
                        id="expiresInSelected"
                        placeholder="Select a duration"
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(EXPIRES_IN_OPTS).map(([label, value]) => (
                        <SelectItem key={value} value={value}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                );
              }}
            />

            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Generate token
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}
