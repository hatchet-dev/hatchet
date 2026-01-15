import { SectionHeader } from './section-header';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { CheckIcon, ChevronRightIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';

const workflowSteps = {
  settingUp: 'Setting Up',
  chooseToken: 'Set env vars',
  runWorker: 'Run worker',
  viewTask: 'Launch task',
};

const workflowLanguages = {
  python: 'Python',
  typescript: 'TypeScript',
  go: 'Go',
};

export type WorkflowStepKey = keyof typeof workflowSteps;
export type WorkflowLanguageKey = keyof typeof workflowLanguages;

export function LearnWorkflowSection({
  tenantId,
  selectedTab,
  onSelectedTabChange,
  language,
  onLanguageChange,
  hasActiveWorker,
  onTabChangeEvent,
  onLanguageSelectedEvent,
  onFinish,
}: {
  tenantId: string;
  selectedTab: WorkflowStepKey;
  onSelectedTabChange: (tab: WorkflowStepKey) => void;
  language: WorkflowLanguageKey;
  onLanguageChange: (language: WorkflowLanguageKey) => void;
  hasActiveWorker: boolean;
  onTabChangeEvent?: (tab: WorkflowStepKey, tabLabel: string) => void;
  onLanguageSelectedEvent?: (
    language: WorkflowLanguageKey,
    label: string,
  ) => void;
  onFinish: () => void;
}) {
  return (
    <div>
      <SectionHeader title="Learn the workflow" showOnboardingBadge />
      <Tabs
        value={selectedTab}
        onValueChange={(value) => {
          const newTab = value as WorkflowStepKey;
          onSelectedTabChange(newTab);
          onTabChangeEvent?.(newTab, workflowSteps[newTab]);
        }}
        className="w-full rounded-md px-6 pb-6 bg-muted/20 ring-1 ring-border/50 ring-inset"
      >
        <TabsList className="grid w-full grid-flow-col rounded-none bg-transparent p-0 pb-6 justify-start gap-6 h-auto ">
          {Object.entries(workflowSteps).map(([key, value], index) => (
            <TabsTrigger
              key={key}
              value={key}
              className={
                'text-xs text-muted-foreground rounded-none pt-2.5 px-1 font-medium border-t border-transparent hover:border-border bg-transparent data-[state=active]:border-primary/50 data-[state=active]:bg-transparent data-[state=active]:shadow-none'
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
            onValueChange={(value) => {
              const next = value as WorkflowLanguageKey;
              onLanguageChange(next);
              onLanguageSelectedEvent?.(next, workflowLanguages[next]);
            }}
            className="w-full"
          >
            <TabsList className="mt-2 bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
              {Object.entries(workflowLanguages).map(([key, value]) => (
                <TabsTrigger
                  key={key}
                  value={key}
                  className="rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset"
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
            onClick={() => onSelectedTabChange('chooseToken')}
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </TabsContent>

        <TabsContent value="chooseToken" className="mt-0 space-y-5">
          <p>
            Set your API token as an environment variable. Copy the token from
            the &quot;Create API token&quot; section above.
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`export HATCHET_CLIENT_TOKEN="your-api-token"`}
            copyCode={`export HATCHET_CLIENT_TOKEN="your-api-token"`}
            language="shell"
            copy
          />
          <p>If running locally, also set:</p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`export HATCHET_CLIENT_TLS_STRATEGY=none`}
            copyCode={`export HATCHET_CLIENT_TLS_STRATEGY=none`}
            language="shell"
            copy
          />
          <Button
            variant="outline"
            size="default"
            className="w-fit gap-2 bg-muted/70"
            onClick={() => onSelectedTabChange('runWorker')}
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </TabsContent>

        <TabsContent value="runWorker" className="mt-0 space-y-5">
          <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
            {hasActiveWorker ? (
              <>
                <CheckIcon className="size-5 text-green-500" />
                <span className="text-sm font-medium">Worker is connected</span>
              </>
            ) : (
              <>
                <Spinner className="size-5" />
                <span className="text-sm text-muted-foreground">
                  Waiting for worker...
                </span>
              </>
            )}
          </div>

          <p className="text-sm text-muted-foreground">
            Start your worker by running the quickstart code in your terminal.
            Once connected, you&apos;ll see a confirmation above.
          </p>

          <Button
            variant="outline"
            size="default"
            className="w-fit gap-2 bg-muted/70"
            onClick={() => onSelectedTabChange('viewTask')}
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </TabsContent>

        <TabsContent value="viewTask" className="mt-0 space-y-5">
          <p className="text-sm">
            Your worker is connected and ready to process tasks. Here&apos;s
            what you can do next:
          </p>

          <div className="space-y-3">
            <Link
              to="/tenants/$tenant/runs"
              params={{ tenant: tenantId }}
              className="flex items-center gap-2 text-sm text-primary hover:underline"
            >
              <ChevronRightIcon className="size-4" />
              Trigger a task
            </Link>
            <Link
              to="/tenants/$tenant/workflows"
              params={{ tenant: tenantId }}
              className="flex items-center gap-2 text-sm text-primary hover:underline"
            >
              <ChevronRightIcon className="size-4" />
              View list of workflows
            </Link>
          </div>

          <Button
            variant="outline"
            size="default"
            className="w-fit gap-2 bg-muted/70"
            onClick={onFinish}
          >
            Finish
            <CheckIcon className="size-3 text-brand" />
          </Button>
        </TabsContent>
      </Tabs>
    </div>
  );
}
