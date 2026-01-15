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
import { TriggerWorkflowForm } from '@/pages/main/v1/workflows/$workflow/components/trigger-workflow-form';
import { CheckIcon, ChevronRightIcon } from '@radix-ui/react-icons';
import { Play } from 'lucide-react';
import { useEffect, useState, type ReactNode } from 'react';

export const workflowStepOptions = {
  install: { value: 'install', label: 'Install the CLI' },
  profile: { value: 'profile', label: 'Set your profile' },
  quickstart: { value: 'quickstart', label: 'Project quickstart' },
  runTask: { value: 'runTask', label: 'Run a task' },
} as const;

export const workflowLanguageOptions = {
  python: { value: 'python', label: 'Python' },
  typescript: { value: 'typescript', label: 'TypeScript' },
  go: { value: 'go', label: 'Go' },
} as const;

export const installMethodOptions = {
  native: { value: 'native', label: 'Native (Recommended)' },
  homebrew: { value: 'homebrew', label: 'Homebrew' },
} as const;

export type WorkflowStepKey = keyof typeof workflowStepOptions;
export type WorkflowLanguageKey =
  (typeof workflowLanguageOptions)[keyof typeof workflowLanguageOptions]['value'];
export type InstallMethod =
  (typeof installMethodOptions)[keyof typeof installMethodOptions]['value'];

export function LearnWorkflowSection({
  tenantName,
  selectedTab,
  onSelectedTabChange,
  profileToken,
  isGeneratingProfileToken,
  profileTokenError,
  onGenerateProfileToken,
  hasActiveWorker,
  onTabChangeEvent,
  onFinish,
  installMethod,
  onInstallMethodChange,
}: {
  tenantName?: string;
  selectedTab: WorkflowStepKey;
  onSelectedTabChange: (tab: WorkflowStepKey) => void;
  language: WorkflowLanguageKey;
  onLanguageChange: (language: WorkflowLanguageKey) => void;
  profileToken?: string;
  isGeneratingProfileToken: boolean;
  profileTokenError?: string;
  onGenerateProfileToken: () => void;
  hasActiveWorker: boolean;
  onTabChangeEvent?: (tab: WorkflowStepKey, tabLabel: string) => void;
  onLanguageSelectedEvent?: (
    language: WorkflowLanguageKey,
    label: string,
  ) => void;
  onFinish: () => void;
  installMethod: InstallMethod;
  onInstallMethodChange: (installMethod: InstallMethod) => void;
}) {
  const profileName = tenantName?.trim() || 'local';
  const escapeForDoubleQuotes = (value: string) =>
    value
      .replace(/\\/g, '\\\\')
      .replace(/"/g, '\\"')
      .replace(/\$/g, '\\$')
      .replace(/`/g, '\\`');

  const [showTriggerWorkflow, setShowTriggerWorkflow] = useState(false);
  const [hasCopiedProfileToken, setHasCopiedProfileToken] = useState(false);

  useEffect(() => {
    setHasCopiedProfileToken(false);
  }, [profileToken]);

  const steps: Array<{
    value: WorkflowStepKey;
    label: string;
    content: ReactNode;
  }> = [
    {
      ...workflowStepOptions.install,
      content: (
        <>
          <p> Install the Hatchet CLI. </p>
          <Tabs
            value={installMethod}
            onValueChange={(value) => {
              onInstallMethodChange(value as InstallMethod);
            }}
            className="w-full"
          >
            <TabsList className="mt-2 bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
              {Object.entries(installMethodOptions).map(([key, value]) => (
                <TabsTrigger
                  key={key}
                  value={value.value}
                  className="rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset"
                >
                  {value.label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent
              value={installMethodOptions.native.value}
              className="mt-4 space-y-3"
            >
              <b>MacOS, Linux, WSL</b>
              <CodeHighlighter
                className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                code={`curl -fsSL https://install.hatchet.run/install.sh | bash`}
                language="shell"
                copy
              />
            </TabsContent>

            <TabsContent
              value={installMethodOptions.homebrew.value}
              className="mt-4 space-y-3"
            >
              <b>MacOS</b>
              <CodeHighlighter
                className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                code={`brew install hatchet-dev/hatchet/hatchet --cask`}
                language="shell"
                copy
              />
            </TabsContent>
          </Tabs>
          <p>Verify the installation by running:</p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet --version`}
            language="shell"
            copy
          />
          <Button
            variant="outline"
            size="default"
            className="w-fit gap-2 bg-muted/70"
            onClick={() =>
              onSelectedTabChange(workflowStepOptions.profile.value)
            }
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </>
      ),
    },
    {
      ...workflowStepOptions.profile,
      content: (
        <>
          <p>Add a Hatchet CLI profile using an API token.</p>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              onClick={() => {
                setHasCopiedProfileToken(false);
                onGenerateProfileToken();
              }}
              disabled={isGeneratingProfileToken}
            >
              {isGeneratingProfileToken && <Spinner />}
              Generate token for this command
            </Button>
            {profileToken && (
              <span className="text-xs text-muted-foreground">
                This token is only shown once — copy it now.
              </span>
            )}
          </div>
          {profileTokenError && (
            <div className="text-sm text-red-500">{profileTokenError}</div>
          )}
          {profileToken && (
            <div
              onMouseDown={() => setHasCopiedProfileToken(true)}
              onClick={() => setHasCopiedProfileToken(true)}
            >
              <CodeHighlighter
                className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                code={`hatchet profile add --name "${escapeForDoubleQuotes(
                  profileName,
                )}" --token "${escapeForDoubleQuotes(profileToken)}"`}
                language="shell"
                copy
                onCopy={() => setHasCopiedProfileToken(true)}
              />
            </div>
          )}
          <div className="flex flex-wrap items-center gap-2 justify-between">
            <Button
              variant="outline"
              size="default"
              className="w-fit gap-2 bg-muted/70"
              disabled={!profileToken || !hasCopiedProfileToken}
              onClick={() =>
                onSelectedTabChange(workflowStepOptions.quickstart.value)
              }
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
            <Button
              variant="ghost"
              size="default"
              className="w-fit"
              onClick={() =>
                onSelectedTabChange(workflowStepOptions.quickstart.value)
              }
            >
              Skip
            </Button>
          </div>
        </>
      ),
    },
    {
      ...workflowStepOptions.quickstart,
      content: (
        <>
          <p>
            Run the quickstart command to clone an example project repository
            and follow the instructions to cd into the project directory..
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet quickstart`}
            language="shell"
            copy
          />

          <p>
            Then, start your worker in development mode. This will start a
            worker that will listen for tasks and run them locally.
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet worker dev`}
            language="shell"
            copy
          />

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
          <Button
            variant="outline"
            size="default"
            className="w-fit gap-2 bg-muted/70"
            onClick={() =>
              onSelectedTabChange(workflowStepOptions.runTask.value)
            }
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </>
      ),
    },
    {
      ...workflowStepOptions.runTask,
      content: (
        <>
          <p className="text-sm">
            Your worker is connected and ready to process tasks. There are
            multiple ways to invoke a task:
          </p>

          <div className="space-y-3">
            <p>1. via the cli:</p>
            <p>
              In a new terminal, run the following command to invoke a task:
            </p>
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={`hatchet task run --input '{"message": "Hello, World!"}'`}
              language="shell"
              copy
            />
            <p>2. via the sdk:</p>
            <p>
              In a new terminal, run the following command to invoke a task:
            </p>
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={`curl -X POST -H "Content-Type: application/json" -d '{"message": "Hello, World!"}' http://localhost:8080/tasks/run`}
              language="shell"
              copy
            />
            <p>3. via the dashboard:</p>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={() => setShowTriggerWorkflow(true)}
                className="w-fit"
              >
                <span className="cq-xl:inline hidden text-sm">Trigger Run</span>
                <Play className="cq-xl:hidden size-4" />
              </Button>
            </div>
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
        </>
      ),
    },
  ];

  return (
    <div>
      <TriggerWorkflowForm
        defaultWorkflow={undefined}
        show={showTriggerWorkflow}
        onClose={() => setShowTriggerWorkflow(false)}
      />
      <SectionHeader title="Setup your local environment" showOnboardingBadge />
      <Tabs
        value={selectedTab}
        onValueChange={(value) => {
          onSelectedTabChange(value as WorkflowStepKey);
          onTabChangeEvent?.(
            value as WorkflowStepKey,
            workflowStepOptions[value as WorkflowStepKey].label,
          );
        }}
        className="w-full rounded-md px-6 pb-6 bg-muted/20 ring-1 ring-border/50 ring-inset"
      >
        <TabsList className="grid w-full grid-flow-col rounded-none bg-transparent p-0 pb-6 justify-start gap-6 h-auto ">
          {steps.map((step, index) => (
            <TabsTrigger
              key={step.value}
              value={step.value}
              className={
                'text-xs text-muted-foreground rounded-none pt-2.5 px-1 font-medium border-t border-transparent hover:border-border bg-transparent data-[state=active]:border-primary/50 data-[state=active]:bg-transparent data-[state=active]:shadow-none'
              }
            >
              <div className="flex items-center gap-2">
                {index + 1} {step.label}
              </div>
            </TabsTrigger>
          ))}
        </TabsList>
        {steps.map((step) => (
          <TabsContent
            key={step.value}
            value={step.value}
            className="mt-0 space-y-5"
          >
            {step.content}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
