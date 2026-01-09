import { OnboardingWidget } from './components/onboarding-widget';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Separator } from '@/components/v1/ui/separator';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { useState } from 'react';

const expiresInOptions = [
  { label: '1 hour', value: '1h' },
  { label: '1 day', value: '1d' },
  { label: '1 week', value: '1w' },
  { label: '1 month', value: '1m' },
  { label: '1 year', value: '1y' },
];

export default function Overview() {
  const [expiresIn, setExpiresIn] = useState(expiresInOptions[0].value);

  return (
    <div className="flex h-full w-full flex-col gap-12 p-6">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-6 flex-wrap">
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          <OnboardingWidget steps={4} currentStep={2} label="Steps completed" />
        </div>
        <p className="text-sm text-muted-foreground">
          Get a quick overview of your workflows, runs, and workers.
        </p>
      </div>
      <div>
        <h2 className="text-md">Create API token</h2>{' '}
        <Separator className="my-4" />
        <div className="grid gap-4 items-end grid-flow-col [grid-template-columns:1fr_1fr_auto_1fr]">
          <div className="grid gap-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              type="text"
              required={true}
              autoCapitalize="none"
              autoCorrect="off"
              placeholder="Tenant Name"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="expiresIn">Expires In</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="flex justify-between data-[state=open]:bg-muted"
                >
                  Expires In{' '}
                  {expiresInOptions.find((option) => option.value === expiresIn)
                    ?.label || 'Select an option'}
                  <ChevronDownIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[160px]">
                {expiresInOptions.map((option) => (
                  <DropdownMenuItem
                    key={option.value}
                    onClick={() => setExpiresIn(option.value)}
                  >
                    {option.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <Separator orientation="vertical" />
          <div className="grid gap-2">
            <Button variant="default" size="sm">
              Generate Token
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
