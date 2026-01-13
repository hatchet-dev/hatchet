import { OnboardingWidget } from './components/onboarding-widget';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/v1/ui/tabs';
import {
  ChevronRightIcon,
  ChevronDownIcon,
  CheckIcon,
} from '@radix-ui/react-icons';
import { useState } from 'react';

const expiresInOptions = [
  { label: '1 hour', value: '1h' },
  { label: '1 day', value: '1d' },
  { label: '1 week', value: '1w' },
  { label: '1 month', value: '1m' },
  { label: '1 year', value: '1y' },
];

const onboardingSVG = (
  <svg
    width="12"
    height="12"
    viewBox="0 0 12 12"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className="bottom-[1px] relative"
  >
    <path
      d="M10.499 1.5V2.5C10.499 7.3137 7.81275 9.5 4.49902 9.5L3.5482 9.49995C3.65395 7.9938 4.1227 7.0824 5.34695 5.99945C5.94895 5.4669 5.89825 5.15945 5.60145 5.33605C3.55965 6.5508 2.54557 8.1931 2.50059 10.8151L2.49902 11H1.49902C1.49902 10.3187 1.55688 9.69985 1.6719 9.1341C1.55665 8.48705 1.49902 7.6088 1.49902 6.5C1.49902 3.73857 3.7376 1.5 6.499 1.5C7.499 1.5 8.499 2 10.499 1.5Z"
      fill="hsl(var(--brand))"
    />
  </svg>
);

const workflowSteps = {
  settingUp: 'Setting Up',
  chooseToken: 'Choose token',
  runWorker: 'Run worker',
  viewTask: 'View task',
};

const workflowLanguages = {
  python: 'Python',
  typescript: 'TypeScript',
  go: 'Go',
};

export default function Overview() {
  const [expiresIn, setExpiresIn] = useState(expiresInOptions[0].value);
  const [selectedTab, setSelectedTab] = useState<
    'settingUp' | 'chooseToken' | 'runWorker' | 'viewTask'
  >('settingUp');
  const [language, setLanguage] = useState<string>('python');

  return (
    <div className="flex h-full w-full flex-col space-y-20 p-6">
      <div className="grid grid-cols-[1fr_auto] gap-2 items-start">
        <div className="flex items-center gap-6 flex-wrap">
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          <OnboardingWidget steps={4} currentStep={1} label="Steps completed" />
        </div>
        <p className="text-muted-foreground text-balance">
          Get a quick overview of your
          <br />
          workflows, runs, and workers.
        </p>
      </div>
      <div>
        <span className="inline-flex items-baseline gap-5">
          <h2 className="text-md">Create API token</h2>
          <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-brand">
            {onboardingSVG}
            Onboarding step
          </span>
        </span>
        <Separator className="my-4 bg-border/50" flush />
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
                  size="default"
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
          <div className="grid gap-2 justify-self-start">
            <Button variant="default" size="default">
              Generate Token
            </Button>
          </div>
        </div>
      </div>
      <div>
        <span className="inline-flex items-baseline gap-5">
          <h2 className="text-md">Learn the workflow</h2>
          <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-brand">
            {onboardingSVG}
            Onboarding step
          </span>
        </span>
        <Separator className="my-4 bg-border/50" flush />
        <Tabs
          value={selectedTab}
          onValueChange={(value) =>
            setSelectedTab(
              value as 'settingUp' | 'chooseToken' | 'runWorker' | 'viewTask',
            )
          }
          className="w-full rounded-md px-6 pb-6 bg-muted/20 ring-1 ring-border/50 ring-inset"
        >
          <TabsList className="grid w-full grid-flow-col rounded-none bg-transparent p-0 pb-6 justify-start gap-6 h-auto">
            {Object.entries(workflowSteps).map(([key, value], index) => (
              <TabsTrigger
                key={key}
                value={key}
                className={
                  'text-xs rounded-none pt-2.5 px-1 font-medium border-t border-transparent hover:border-border bg-transparent data-[state=active]:border-primary/50 data-[state=active]:bg-transparent'
                }
              >
                <div className="flex items-center gap-2">
                  {index + 1} {value}
                </div>
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="settingUp" className="mt-0 space-y-5">
            <p> Clone the repository and install dependencies. </p>
            <Tabs
              value={language}
              onValueChange={setLanguage}
              className="w-full"
            >
              <TabsList className="mt-2 bg-muted/20 ring-1 ring-border/50 ring-inset rounded-lg p-0 gap-0.5">
                {Object.entries(workflowLanguages).map(([key, value]) => (
                  <TabsTrigger
                    key={key}
                    value={key}
                    className="rounded-lg h-full ring-inset data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-muted/70"
                  >
                    {value}
                  </TabsTrigger>
                ))}
              </TabsList>
              <TabsContent value="python" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\npoetry install`}
                  language="shell"
                  copy
                />
              </TabsContent>
              <TabsContent value="typescript" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart`}
                  language="shell"
                  copy
                />
              </TabsContent>
              <TabsContent value="go" className="mt-4 space-y-3">
                <CodeHighlighter
                  className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                  code={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                  copyCode={`git clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\ngo mod tidy`}
                  language="shell"
                  copy
                />
              </TabsContent>
            </Tabs>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('chooseToken')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>

          <TabsContent value="chooseToken" className="mt-0 space-y-5">
            <div> Choose token </div>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('runWorker')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>
          <TabsContent value="runWorker" className="mt-0 space-y-5">
            <div> Run worker </div>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => setSelectedTab('viewTask')}
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
          </TabsContent>
          <TabsContent value="viewTask" className="mt-0 space-y-5">
            <div> View task </div>
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => ({})}
            >
              Finish
              <CheckIcon className="size-3 text-brand" />
            </Button>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
