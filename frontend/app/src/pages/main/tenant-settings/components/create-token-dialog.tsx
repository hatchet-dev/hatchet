import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Code } from '@/components/ui/code';
import { Button } from '@/components/ui/button';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/ui/loading';

const schema = z.object({
  name: z.string().min(1).max(255),
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
        <Code
          language="typescript"
          className="text-sm"
          wrapLines={false}
          maxWidth={'calc(700px - 4rem)'}
          copy
        >
          {token}
        </Code>
      </DialogContent>
    );
  }

  // TODO: add a name for the token

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
