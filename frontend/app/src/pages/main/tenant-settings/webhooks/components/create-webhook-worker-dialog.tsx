import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { z } from 'zod';
import { useFieldArray, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { Spinner } from '@/components/ui/loading';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover.tsx';
import { CheckIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import { Separator } from '@/components/ui/separator.tsx';
import { Badge } from '@/components/ui/badge.tsx';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command.tsx';
import { useEffect, useState } from 'react';
import { Workflow } from '@/lib/api';

const schema = z.object({
  name: z.string().min(1).max(255).optional(),
  url: z.string().url().min(1).max(255),
  secret: z.string().min(1).max(255).optional(),
  workflows: z.array(z.string().uuid()),
});

interface CreateTokenDialogProps {
  workflows: Workflow[];
  className?: string;
  token?: string;
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  fieldErrors?: Record<string, string>;
}

export function CreateWebhookWorkerDialog({
  className,
  token,
  workflows,
  ...props
}: CreateTokenDialogProps) {
  const {
    control,
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      workflows: [],
    },
  });

  const nameError = errors.name?.message?.toString() || props.fieldErrors?.name;
  const urlError = errors.url?.message?.toString() || props.fieldErrors?.url;

  if (token) {
    return (
      <DialogContent className="w-fit max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Keep it secret, keep it safe</DialogTitle>
        </DialogHeader>
        <p className="text-sm">
          Copy the webhook secret and add it in your application.
        </p>
        <CodeHighlighter
          language="typescript"
          className="text-sm"
          wrapLines={false}
          maxWidth={'calc(700px - 4rem)'}
          code={token}
          copy
        />
      </DialogContent>
    );
  }

  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create a new Webhook Endpoint</DialogTitle>
      </DialogHeader>
      <div className={cn('grid gap-6', className)}>
        <form
          onSubmit={handleSubmit((d) => {
            props.onSubmit(d);
          })}
        >
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                {...register('name')}
                id="api-token-name"
                name="name"
                placeholder="My Webhook Endpoint"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {nameError && (
                <div className="text-sm text-red-500">{nameError}</div>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="url">URL</Label>
              <Input
                {...register('url')}
                id="api-token-url"
                name="url"
                placeholder="The Webhook URL"
                autoCapitalize="none"
                autoCorrect="off"
                disabled={props.isLoading}
              />
              {urlError && (
                <div className="text-sm text-red-500">{urlError}</div>
              )}
            </div>

            <Select workflows={workflows} control={control} />

            <Button disabled={props.isLoading}>
              {props.isLoading && <Spinner />}
              Create
            </Button>
          </div>
        </form>
      </div>
    </DialogContent>
  );
}

function Select({
  workflows,
  control,
}: {
  workflows: Workflow[];
  control: any;
}) {
  const { replace } = useFieldArray({
    control, // control props comes from useForm (optional: if you are using FormProvider)
    name: 'workflows', // name of the field (required)
  });

  const [filterValue, setFilterValue] = useState<string[]>([]);
  const selectedValues = new Set(filterValue);
  const options = workflows.map((workflow) => ({
    label: workflow.name,
    value: workflow.metadata.id,
  }));

  useEffect(() => {
    replace(filterValue);
  }, [filterValue, replace]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <PlusCircledIcon className="mr-2 h-4 w-4" />
          Workflows
          {selectedValues?.size > 0 && (
            <>
              <Separator orientation="vertical" className="mx-2 h-4" />
              <Badge
                variant="secondary"
                className="rounded-sm px-1 font-normal lg:hidden"
              >
                {selectedValues.size}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                <Badge
                  variant="secondary"
                  className="rounded-sm px-1 font-normal"
                >
                  {selectedValues.size} selected
                </Badge>
              </div>
            </>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[300px] p-2" align="start">
        <Command>
          <CommandInput placeholder={'Workflows'} />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup>
              {options?.map((option) => {
                const isSelected = selectedValues.has(option.value);
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={() => {
                      if (isSelected) {
                        selectedValues.delete(option.value);
                      } else {
                        selectedValues.add(option.value);
                      }
                      const filterValues = Array.from(selectedValues);
                      setFilterValue(filterValues);
                    }}
                  >
                    <div
                      className={cn(
                        'mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                        isSelected
                          ? 'bg-primary text-primary-foreground'
                          : 'opacity-50 [&_svg]:invisible',
                      )}
                    >
                      <CheckIcon className={cn('h-4 w-4')} />
                    </div>
                    <span>{option.label}</span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
            {selectedValues.size > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={() => setFilterValue([])}
                    className="justify-center text-center"
                  >
                    Reset
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
