import {
  installMethodOptions,
  workflowLanguageOptions,
  workflowStepOptions,
  type InstallMethod,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
} from './onboarding-options';
import { SectionHeader } from './section-header';
import {
  availableUseCases,
  isLanguageSupported,
  escapeForDoubleQuotes,
  scaffoldCommand,
  triggerCommand,
  workerDevCommand,
  type AvailableUseCaseKey,
} from './use-case-options';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import { RadioGroup, RadioGroupCardItem } from '@/components/v1/ui/radio-group';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import useCanWrite from '@/hooks/use-can-write';
import { TriggerWorkflowForm } from '@/pages/main/v1/workflows/$workflow/components/trigger-workflow-form';
import { CheckIcon, ChevronRightIcon } from '@radix-ui/react-icons';
import { useState, type ReactNode } from 'react';

// Re-exported so existing importers keep working after the catalogs moved
// to onboarding-options.ts.
export {
  installMethodOptions,
  workflowLanguageOptions,
  workflowStepOptions,
  type InstallMethod,
  type WorkflowLanguageKey,
  type WorkflowStepKey,
} from './onboarding-options';

export function LearnWorkflowSection({
  tenantName,
  selectedTab,
  onSelectedTabChange,
  useCase,
  onUseCaseChange,
  language,
  onLanguageChange,
  profileToken,
  isGeneratingProfileToken,
  profileTokenError,
  onGenerateProfileToken,
  hasConnectedWorker,
  hasQualifiedRun,
  onViewRuns,
  onSkip,
  onTabChangeEvent,
  onLanguageSelectedEvent,
  onUseCaseSelectedEvent,
  onFinish,
  installMethod,
  onInstallMethodChange,
  authDisabled,
  authDisabledToken,
}: {
  tenantName?: string;
  selectedTab: WorkflowStepKey;
  onSelectedTabChange: (tab: WorkflowStepKey) => void;
  useCase: AvailableUseCaseKey;
  onUseCaseChange: (useCase: AvailableUseCaseKey) => void;
  language: WorkflowLanguageKey;
  onLanguageChange: (language: WorkflowLanguageKey) => void;
  profileToken?: string;
  isGeneratingProfileToken: boolean;
  profileTokenError?: string;
  onGenerateProfileToken: () => void;
  // An ACTIVE worker registered after the confirmed selection exists.
  hasConnectedWorker: boolean;
  // A completed run created after the confirmed selection exists.
  hasQualifiedRun: boolean;
  onViewRuns: () => void;
  onSkip: () => void;
  onTabChangeEvent?: (tab: WorkflowStepKey, tabLabel: string) => void;
  onLanguageSelectedEvent?: (
    language: WorkflowLanguageKey,
    label: string,
  ) => void;
  onUseCaseSelectedEvent?: (
    useCase: AvailableUseCaseKey,
    label: string,
  ) => void;
  onFinish: () => void;
  installMethod: InstallMethod;
  onInstallMethodChange: (installMethod: InstallMethod) => void;
  authDisabled?: boolean;
  authDisabledToken?: string;
}) {
  const canWrite = useCanWrite();
  const profileName = tenantName?.trim() || 'local';

  const [showTriggerWorkflow, setShowTriggerWorkflow] = useState(false);

  // The shared Button removes the native focus outline without a
  // replacement, so every focusable onboarding control carries an explicit
  // ring here rather than changing the shared primitives app-wide.
  const focusRing =
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background';

  const steps: Array<{
    value: WorkflowStepKey;
    label: string;
    content: ReactNode;
  }> = [
    {
      ...workflowStepOptions.chooseUseCase,
      content: (
        <>
          <p className="text-sm">Preferred language</p>
          <Tabs
            value={language}
            onValueChange={(value) => {
              const nextLanguage = value as WorkflowLanguageKey;
              onLanguageChange(nextLanguage);
              onLanguageSelectedEvent?.(
                nextLanguage,
                workflowLanguageOptions[nextLanguage].label,
              );
            }}
            className="w-full"
          >
            <TabsList className="mt-2 bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
              {Object.values(workflowLanguageOptions).map((option) => (
                <TabsTrigger
                  key={option.value}
                  value={option.value}
                  disabled={!isLanguageSupported(useCase, option.value)}
                  className={`rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`}
                >
                  {option.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
          <p className="text-sm">Use case</p>
          <RadioGroup
            value={useCase}
            onValueChange={(value) => {
              const nextUseCase = value as AvailableUseCaseKey;
              onUseCaseChange(nextUseCase);
              onUseCaseSelectedEvent?.(
                nextUseCase,
                availableUseCases[nextUseCase].label,
              );
            }}
            className="grid-cols-1 gap-3 lg:grid-cols-2"
          >
            {Object.values(availableUseCases).map((option) => (
              <RadioGroupCardItem key={option.value} value={option.value}>
                <span className="block text-sm font-medium">
                  {option.label}
                </span>
                <span className="mt-1 block text-sm text-muted-foreground">
                  {option.description}
                </span>
              </RadioGroupCardItem>
            ))}
          </RadioGroup>
          <Button
            variant="outline"
            size="default"
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
            onClick={() =>
              onSelectedTabChange(workflowStepOptions.install.value)
            }
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </>
      ),
    },
    {
      ...workflowStepOptions.install,
      content: (
        <>
          <p className="text-sm"> Install the Hatchet CLI. </p>
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
                  className={`rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`}
                >
                  {value.label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent
              value={installMethodOptions.native.value}
              className={`mt-4 space-y-3 rounded-sm ${focusRing}`}
            >
              <p className="text-sm">
                <b>MacOS, Linux, WSL</b>
              </p>
              <CodeHighlighter
                className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                code={`curl -fsSL https://install.hatchet.run/install.sh | bash`}
                language="shell"
                copy
              />
            </TabsContent>

            <TabsContent
              value={installMethodOptions.homebrew.value}
              className={`mt-4 space-y-3 rounded-sm ${focusRing}`}
            >
              <p className="text-sm">
                <b>MacOS</b>
              </p>
              <CodeHighlighter
                className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
                code={`brew install hatchet-dev/hatchet/hatchet --cask`}
                language="shell"
                copy
              />
            </TabsContent>
          </Tabs>
          <p className="text-sm">Verify the installation by running:</p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet --version`}
            language="shell"
            copy
          />
          <Button
            variant="outline"
            size="default"
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
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
      content: authDisabled ? (
        <>
          <p className="text-sm">
            Authentication is disabled, but workers still authenticate over gRPC
            with an API token.
          </p>
          <p className="text-sm">
            This instance ships with a built-in token for the default tenant.
            Add it to a CLI profile:
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet profile add --name "${escapeForDoubleQuotes(
              profileName,
            )}" --token "${authDisabledToken ?? '<token>'}"`}
            language="shell"
            copy
          />
          <Button
            variant="outline"
            size="default"
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
            onClick={() =>
              onSelectedTabChange(workflowStepOptions.quickstart.value)
            }
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </>
      ) : (
        <>
          <p className="text-sm">
            Add a Hatchet CLI profile using an API token.
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              variant="outline"
              size="default"
              className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
              onClick={onGenerateProfileToken}
              disabled={isGeneratingProfileToken || !canWrite}
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
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={`hatchet profile add --name "${escapeForDoubleQuotes(
                profileName,
              )}" --token "${escapeForDoubleQuotes(profileToken)}"`}
              language="shell"
              copy
            />
          )}
          <p className="text-sm text-muted-foreground">
            Already have a Hatchet CLI profile? Continue to the next step.
          </p>
          <Button
            variant="outline"
            size="default"
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
            onClick={() =>
              onSelectedTabChange(workflowStepOptions.quickstart.value)
            }
          >
            Continue
            <ChevronRightIcon className="size-3 text-foreground/50" />
          </Button>
        </>
      ),
    },
    {
      ...workflowStepOptions.quickstart,
      content: (
        <>
          <p className="text-sm">
            Run the quickstart command to generate an example project for your
            selected use case. The CLI asks for the package manager, project
            name, and directory.
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={scaffoldCommand({ useCase, language })}
            language="shell"
            copy
          />

          <p className="text-sm">
            Then, start your worker in development mode. This will start a
            worker that will listen for tasks and run them locally.
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={workerDevCommand(profileName)}
            language="shell"
            copy
          />

          <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
            {hasConnectedWorker ? (
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
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
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
            With the worker running, you can now open a new terminal and run the
            following command to trigger a task run:
          </p>

          <div className="space-y-3">
            <CodeHighlighter
              className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
              code={triggerCommand(useCase, profileName)}
              language="shell"
              copy
            />
            <p className="text-sm">
              You can view the script to understand how to trigger a task run
              from your own codebase.
            </p>
          </div>

          <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
            {hasQualifiedRun ? (
              <>
                <CheckIcon className="size-5 text-green-500" />
                <span className="text-sm font-medium">Run completed</span>
              </>
            ) : (
              <>
                <Spinner className="size-5" />
                <span className="text-sm text-muted-foreground">
                  Waiting for a completed run...
                </span>
              </>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2 justify-between">
            <Button
              variant="outline"
              size="default"
              className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
              disabled={!hasQualifiedRun}
              onClick={() =>
                onSelectedTabChange(workflowStepOptions.aiDocs.value)
              }
            >
              Continue
              <ChevronRightIcon className="size-3 text-foreground/50" />
            </Button>
            <Button
              variant="ghost"
              size="default"
              className={`w-fit ${focusRing}`}
              onClick={onViewRuns}
            >
              View runs
            </Button>
          </div>
        </>
      ),
    },
    {
      ...workflowStepOptions.aiDocs,
      content: (
        <>
          <p className="text-sm">
            Optional next step: get Hatchet documentation directly in your AI
            coding assistant (Cursor, Claude Code, Claude Desktop, and more).
          </p>
          <CodeHighlighter
            className="bg-muted/20 ring-1 ring-border/50 ring-inset px-1"
            code={`hatchet docs install`}
            language="shell"
            copy
          />
          <p className="text-sm text-muted-foreground">
            See the{' '}
            <a
              href="https://docs.hatchet.run/home/install-docs-mcp"
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:text-foreground"
            >
              full setup guide
            </a>{' '}
            for manual configuration options.
          </p>
          {!hasQualifiedRun && (
            <p className="text-sm text-muted-foreground">
              Waiting for a completed run before finishing. See the Run a task
              step.
            </p>
          )}
          <Button
            variant="outline"
            size="default"
            className={`w-fit gap-2 bg-muted/70 ${focusRing}`}
            disabled={!hasQualifiedRun}
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
      <SectionHeader
        title="Set up your local environment"
        showOnboardingBadge
      />
      <div className="mb-2 flex justify-end">
        <Button
          variant="ghost"
          size="sm"
          className={`text-muted-foreground ${focusRing}`}
          onClick={onSkip}
        >
          Skip onboarding
        </Button>
      </div>
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
              className={`text-xs text-muted-foreground rounded-none pt-2.5 px-1 font-medium border-t border-transparent hover:border-border bg-transparent data-[state=active]:border-primary/50 data-[state=active]:bg-transparent data-[state=active]:shadow-none ${focusRing}`}
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
            className={`mt-0 space-y-5 rounded-sm ${focusRing}`}
          >
            {step.content}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
