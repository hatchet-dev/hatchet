import {
  installMethodOptions,
  workflowLanguageOptions,
  type InstallMethod,
  type WorkflowLanguageKey,
} from './onboarding-options';
import {
  buildOnboardingPrompt,
  type AgentUseCaseKey,
} from './prompt-templates';
import {
  availableUseCases,
  escapeForDoubleQuotes,
  scaffoldCommand,
  triggerCommand,
  workerDevCommand,
  type AvailableUseCaseKey,
} from './use-case-options';
import { type OnboardingProgress } from './use-onboarding-progress';
import { SdkSwitcher, type Sdk } from './use-preferred-sdk';
import { HelpDropdown } from '@/components/v1/nav/help-dropdown';
import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Spinner } from '@/components/v1/ui/loading';
import { RadioGroup, RadioGroupCardItem } from '@/components/v1/ui/radio-group';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { Textarea } from '@/components/v1/ui/textarea';
import { cn } from '@/lib/utils';
import {
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
} from '@radix-ui/react-icons';
import { LifeBuoy } from 'lucide-react';
import { useMemo, useState, type ReactNode } from 'react';

// The first release of the CLI that ships `hatchet profile env` and the
// multi-target `hatchet mcp install`. Both the agent-path commands here and
// the generated prompt depend on it.
// TODO: set once belanger/profile-env releases.
const MIN_CLI_VERSION = 'TBD';

// The shared Button strips the native focus outline without a replacement, so
// each focusable onboarding control carries an explicit ring, matching
// learn-workflow-section rather than changing the shared primitives app-wide.
const focusRing =
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background';

const continueButtonClass = `w-fit gap-2 bg-muted/70 ${focusRing}`;

// A use case, or the sentinel for the freeform "describe your own" choice.
// The agent path offers the full AgentUseCaseKey union plus custom; the manual
// path narrows to the scaffoldable subset (see isScaffoldableUseCase).
type UseCaseChoice = AgentUseCaseKey | 'custom';

// The chosen setup path. It is picked first; null means the path selector is
// still showing and no other step exists yet.
type SetupPath = 'agent' | 'manual';

// True for use cases the CLI can scaffold + trigger (the manual path). The
// agent path additionally offers roadmap use cases (fanout/event/durable) and
// the freeform custom choice, which drive prompt generation only.
function isScaffoldableUseCase(
  choice: UseCaseChoice,
): choice is AvailableUseCaseKey {
  return choice !== 'custom' && choice in availableUseCases;
}

// Agent-path use-case cards. Labels + descriptions from copy 4.1
// (usecase.*.label / .desc). The agent path is prompt-only, so it offers the
// full list including the roadmap use cases the CLI cannot scaffold yet.
const agentUseCaseOptions: {
  value: AgentUseCaseKey;
  label: string;
  description: string;
}[] = [
  {
    value: 'simple',
    label: 'Simple task',
    description: 'A single task that takes an input and returns a result.',
  },
  {
    value: 'scheduled',
    label: 'Cron job / scheduled run',
    description: 'A workflow that runs on a schedule.',
  },
  {
    value: 'fanout',
    label: 'Fan-out / parallel',
    description:
      'A parent task that spawns work in parallel and aggregates the results.',
  },
  {
    value: 'event',
    label: 'Event-driven',
    description: 'A workflow triggered by an event you push.',
  },
  {
    value: 'durable',
    label: 'Durable / long-running',
    description:
      'A task that sleeps or waits for an event and survives restarts.',
  },
];

// Human label for any agent use case, used in the prompt-step body.
function useCaseChoiceLabel(choice: UseCaseChoice): string {
  if (choice === 'custom') {
    return 'use case';
  }
  return (
    agentUseCaseOptions
      .find((option) => option.value === choice)
      ?.label.toLowerCase() ?? 'use case'
  );
}

// Coding agents offered in the MCP multi-select. `value` is the CLI target
// token passed to `hatchet mcp install --target`.
const mcpAgentOptions = [
  { value: 'claude-code', label: 'Claude Code' },
  { value: 'cursor', label: 'Cursor' },
  { value: 'vscode', label: 'VS Code' },
  { value: 'codex', label: 'Codex' },
] as const;

type McpAgentValue = (typeof mcpAgentOptions)[number]['value'];

type StepKey =
  | 'usecase'
  | 'cli'
  | 'profile'
  | 'path'
  | 'mcp'
  | 'prompt'
  | 'quickstart'
  | 'runtask'
  | 'finish';

const stepRailLabels: Record<StepKey, string> = {
  usecase: 'Use case & SDK',
  cli: 'Install CLI',
  profile: 'Profile',
  path: 'Choose path',
  mcp: 'Connect agent',
  prompt: 'Generate prompt',
  quickstart: 'Quickstart',
  runtask: 'Run a task',
  finish: 'Finish',
};

// SDK -> command-builder language. The command builders only speak the three
// fully supported languages; Ruby has no scaffold/trigger template, so the
// manual path is unavailable for it (callers show a docs note instead).
function sdkToLanguage(sdk: Sdk): WorkflowLanguageKey | null {
  switch (sdk) {
    case 'python':
      return workflowLanguageOptions.python.value;
    case 'typescript':
      return workflowLanguageOptions.typescript.value;
    case 'go':
      return workflowLanguageOptions.go.value;
    case 'ruby':
      return null;
  }
}

export function OnboardingSteps({
  tenantName,
  sdk,
  onSdkChange,
  useCase,
  onUseCaseChange,
  onConfirmSelection,
  profileToken,
  isGeneratingProfileToken,
  profileTokenError,
  onGenerateProfileToken,
  canGenerateToken,
  authDisabled,
  authDisabledToken,
  progress,
  onFinish,
  onPromptGenerated,
  onStepChangeEvent,
}: {
  tenantName?: string;
  sdk: Sdk;
  // Updates the global SDK preference; the modal also syncs the persisted
  // onboarding language and analytics.
  onSdkChange: (next: Sdk) => void;
  // The persisted use case (a real, scaffoldable one). The freeform "custom"
  // choice is local and does not touch this.
  useCase: AvailableUseCaseKey;
  onUseCaseChange: (next: AvailableUseCaseKey) => void;
  // Records selectionConfirmedAt (via applyTabChange semantics) so progress
  // polling can begin. Idempotent: only the first call past step 1 sets it.
  onConfirmSelection: () => void;
  profileToken?: string;
  isGeneratingProfileToken: boolean;
  profileTokenError?: string;
  onGenerateProfileToken: () => void;
  canGenerateToken: boolean;
  authDisabled?: boolean;
  authDisabledToken?: string;
  progress: OnboardingProgress;
  onFinish: () => void;
  // Fired when the agent-path prompt is generated or regenerated.
  onPromptGenerated: (template: UseCaseChoice, sdk: Sdk) => void;
  // Fired on navigation to a different step (the stepper analog of a tab
  // change).
  onStepChangeEvent?: (step: StepKey, label: string) => void;
}) {
  const profileName = tenantName?.trim() || 'local';
  const language = sdkToLanguage(sdk);

  const [currentStep, setCurrentStep] = useState<StepKey>('path');
  const [installMethod, setInstallMethod] = useState<InstallMethod>(
    installMethodOptions.native.value,
  );
  // Local-only session state this increment (not added to the persisted
  // onboarding schema): the freeform choice + text, the chosen path, and the
  // selected MCP agents.
  const [useCaseChoice, setUseCaseChoice] = useState<UseCaseChoice>(useCase);
  const [freeform, setFreeform] = useState('');
  const [path, setPath] = useState<SetupPath | null>(null);
  const [selectedAgents, setSelectedAgents] = useState<McpAgentValue[]>([]);

  // The visible step sequence. The path is chosen first, so before a path
  // exists the selector is the only step; picking one reveals the rest.
  const sequence = useMemo<StepKey[]>(() => {
    if (path === 'agent') {
      return ['path', 'usecase', 'cli', 'profile', 'mcp', 'prompt', 'finish'];
    }
    if (path === 'manual') {
      return [
        'path',
        'usecase',
        'cli',
        'profile',
        'quickstart',
        'runtask',
        'finish',
      ];
    }
    return ['path'];
  }, [path]);

  // Confirm the selection once the user advances past the use-case step, so
  // progress polling begins then (never on the first steps). confirmSelection
  // is idempotent, so firing on every later step is safe.
  const goTo = (step: StepKey) => {
    if (step !== 'path' && step !== 'usecase') {
      onConfirmSelection();
    }
    if (step !== currentStep) {
      onStepChangeEvent?.(step, stepRailLabels[step]);
    }
    setCurrentStep(step);
  };

  const choosePath = (next: SetupPath) => {
    setPath(next);
    // Roadmap/custom use cases are agent-only. Switching to the manual path
    // with one selected falls back to the persisted scaffoldable use case.
    if (next === 'manual' && !isScaffoldableUseCase(useCaseChoice)) {
      setUseCaseChoice(useCase);
    }
    goTo('usecase');
  };

  const mcpTargets = selectedAgents.join(',');
  const mcpCommand = `hatchet mcp install --target ${mcpTargets}`;

  const generatedPrompt = useMemo(
    () =>
      buildOnboardingPrompt({
        sdk,
        useCaseKey: useCaseChoice,
        freeform,
      }),
    [sdk, useCaseChoice, freeform],
  );

  const toggleAgent = (value: McpAgentValue) => {
    setSelectedAgents((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    );
  };

  const StatusRow = ({
    done,
    waiting,
    ready,
  }: {
    done: string;
    waiting: string;
    ready: boolean;
  }) => (
    <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-4">
      {ready ? (
        <>
          <CheckIcon className="size-5 text-green-500" />
          <span className="text-sm font-medium">{done}</span>
        </>
      ) : (
        <>
          <Spinner className="size-5" />
          <span className="text-sm text-muted-foreground">{waiting}</span>
        </>
      )}
    </div>
  );

  const codeBlockClass = 'bg-muted/20 ring-1 ring-border/50 ring-inset px-1';

  const stepContent: Record<StepKey, ReactNode> = {
    usecase: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">What are you building?</h3>
          <p className="text-sm text-muted-foreground">
            We'll customize your prompt for getting your coding agent onboarded.
          </p>
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium">SDK</p>
          <SdkSwitcher value={sdk} onChange={onSdkChange} />
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium">Use case</p>
          <RadioGroup
            value={useCaseChoice}
            onValueChange={(value) => {
              const next = value as UseCaseChoice;
              setUseCaseChoice(next);
              // Only scaffoldable use cases feed the manual command builders,
              // so only they update the persisted state.
              if (isScaffoldableUseCase(next)) {
                onUseCaseChange(next);
              }
            }}
            className="grid-cols-1 gap-3 lg:grid-cols-2"
          >
            {path === 'manual'
              ? Object.values(availableUseCases).map((option) => (
                  <RadioGroupCardItem key={option.value} value={option.value}>
                    <span className="block text-sm font-medium">
                      {option.label}
                    </span>
                    <span className="mt-1 block text-sm text-muted-foreground">
                      {option.description}
                    </span>
                  </RadioGroupCardItem>
                ))
              : agentUseCaseOptions.map((option) => (
                  <RadioGroupCardItem key={option.value} value={option.value}>
                    <span className="block text-sm font-medium">
                      {option.label}
                    </span>
                    <span className="mt-1 block text-sm text-muted-foreground">
                      {option.description}
                    </span>
                  </RadioGroupCardItem>
                ))}
            {path !== 'manual' && (
              <RadioGroupCardItem value="custom" className="lg:col-span-2">
                <span className="block text-sm font-medium">
                  Describe your own
                </span>
                <span className="mt-1 block text-sm text-muted-foreground">
                  Tell your agent exactly what to build.
                </span>
              </RadioGroupCardItem>
            )}
          </RadioGroup>
          {path !== 'manual' && useCaseChoice === 'custom' && (
            <Textarea
              value={freeform}
              onChange={(e) => setFreeform(e.target.value)}
              placeholder="e.g. process uploaded CSVs and email a summary when done"
              className={focusRing}
            />
          )}
        </div>
      </>
    ),
    cli: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Install the Hatchet CLI</h3>
          <p className="text-sm text-muted-foreground">
            The CLI sets up your profile, installs the MCP server, and lets your
            agent operate Hatchet.
          </p>
        </div>
        <Tabs
          value={installMethod}
          onValueChange={(value) => setInstallMethod(value as InstallMethod)}
          className="w-full"
        >
          <TabsList className="mt-2 bg-muted ring-1 ring-border/50 rounded-lg p-0 gap-0.5 dark:bg-muted/20 dark:ring-inset">
            <TabsTrigger
              value={installMethodOptions.native.value}
              className={`rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`}
            >
              curl
            </TabsTrigger>
            <TabsTrigger
              value={installMethodOptions.homebrew.value}
              className={`rounded-lg h-full text-muted-foreground data-[state=active]:ring-1 data-[state=active]:ring-border data-[state=active]:bg-background dark:data-[state=active]:bg-muted/70 dark:data-[state=active]:shadow-lg dark:ring-inset ${focusRing}`}
            >
              Homebrew
            </TabsTrigger>
          </TabsList>
          <TabsContent
            value={installMethodOptions.native.value}
            className={`mt-4 space-y-3 rounded-sm ${focusRing}`}
          >
            <p className="text-sm">
              <b>MacOS, Linux, WSL</b>
            </p>
            <CodeHighlighter
              className={codeBlockClass}
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
              className={codeBlockClass}
              code={`brew install hatchet-dev/hatchet/hatchet --cask`}
              language="shell"
              copy
            />
          </TabsContent>
        </Tabs>
        <p className="text-sm">Verify it installed:</p>
        <CodeHighlighter
          className={codeBlockClass}
          code={`hatchet --version`}
          language="shell"
          copy
        />
        <p className="text-sm text-muted-foreground">
          You should see version {MIN_CLI_VERSION} or newer.
        </p>
        <p className="text-sm text-muted-foreground">
          Requires Hatchet CLI {MIN_CLI_VERSION} or newer (for{' '}
          <code>mcp install</code> and <code>profile env</code>). The command
          above always installs the latest.
        </p>
        <p className="text-sm text-muted-foreground">
          Already have an older CLI? Re-run the command above to upgrade.
        </p>
      </>
    ),
    profile: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Set up your profile</h3>
          <p className="text-sm text-muted-foreground">
            A profile connects the CLI to this tenant.
          </p>
        </div>
        {authDisabled ? (
          <>
            <p className="text-sm">
              Auth is disabled on this instance, so use the built-in token
              below.
            </p>
            <CodeHighlighter
              className={codeBlockClass}
              code={`hatchet profile add --name "${escapeForDoubleQuotes(
                profileName,
              )}" --token "${authDisabledToken ?? '<token>'}"`}
              language="shell"
              copy
            />
          </>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <Button
                variant="outline"
                size="default"
                className={continueButtonClass}
                onClick={onGenerateProfileToken}
                disabled={isGeneratingProfileToken || !canGenerateToken}
              >
                {isGeneratingProfileToken && <Spinner />}
                Generate a token for this command
              </Button>
              {profileToken && (
                <span className="text-xs text-muted-foreground">
                  This token is only shown once. Copy it now.
                </span>
              )}
            </div>
            {profileTokenError && (
              <div className="text-sm text-red-500">{profileTokenError}</div>
            )}
            {profileToken && (
              <>
                <p className="text-sm">Then run:</p>
                <CodeHighlighter
                  className={codeBlockClass}
                  code={`hatchet profile add --name "${escapeForDoubleQuotes(
                    profileName,
                  )}" --token "${escapeForDoubleQuotes(profileToken)}"`}
                  language="shell"
                  copy
                />
              </>
            )}
          </>
        )}
      </>
    ),
    path: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">
            Choose your preferred setup
          </h3>
        </div>
        <div className="rounded-lg border border-primary/50 bg-muted/40 p-5 space-y-3">
          <div className="space-y-1">
            <h4 className="text-sm font-semibold">
              With your coding agent (recommended)
            </h4>
            <p className="text-sm text-muted-foreground">
              Connect the Hatchet MCP and let your agent scaffold, run, and
              debug for you.
            </p>
          </div>
          <Button
            variant="default"
            size="default"
            className={`w-fit gap-2 ${focusRing}`}
            onClick={() => choosePath('agent')}
          >
            Set up my agent
            <ChevronRightIcon className="size-3" />
          </Button>
        </div>
        <div className="rounded-lg border border-border/50 bg-muted/20 p-5 space-y-3">
          <div className="space-y-1">
            <h4 className="text-sm font-semibold">Manual setup</h4>
            <p className="text-sm text-muted-foreground">
              Scaffold a project, start a worker, and run a task yourself.
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className={`w-fit px-0 text-muted-foreground underline ${focusRing}`}
            onClick={() => choosePath('manual')}
          >
            Prefer to set it up by hand?
          </Button>
        </div>
      </>
    ),
    mcp: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Connect your coding agent</h3>
          <p className="text-sm text-muted-foreground">
            The Hatchet MCP server lets your agent trigger runs, inspect
            results, and debug workers while it builds with you.
          </p>
        </div>
        <div className="space-y-1">
          <p className="text-sm font-medium">Which coding agents do you use?</p>
          <p className="text-sm text-muted-foreground">
            Pick one or more. We will build the install command for you.
          </p>
        </div>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {mcpAgentOptions.map((option) => {
            const checked = selectedAgents.includes(option.value);
            return (
              <label
                key={option.value}
                className={cn(
                  'flex cursor-pointer items-center gap-3 rounded-lg border border-border/50 bg-muted/20 p-3 text-sm hover:border-border',
                  checked && 'border-primary bg-muted/40',
                )}
              >
                <Checkbox
                  checked={checked}
                  onCheckedChange={() => toggleAgent(option.value)}
                  className={focusRing}
                />
                <span className="font-medium">{option.label}</span>
              </label>
            );
          })}
        </div>
        <p className="text-sm">Run this to connect your agents:</p>
        {selectedAgents.length > 0 ? (
          <CodeHighlighter
            className={codeBlockClass}
            code={mcpCommand}
            language="shell"
            copy
          />
        ) : (
          <p className="text-sm text-muted-foreground">
            Select at least one agent from the list above.
          </p>
        )}
        <a
          href="https://docs.hatchet.run/reference/cli/mcp"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex w-fit items-center gap-1 text-sm underline hover:text-foreground"
        >
          Learn more about the Hatchet MCP
          <ExternalLinkIcon className="size-3" />
        </a>
      </>
    ),
    prompt: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">
            Generate your build prompt
          </h3>
          <p className="text-sm text-muted-foreground">
            Paste this into your coding agent. It tells the agent to connect to
            your live Hatchet instance, use the MCP and the markdown docs, and
            build your {useCaseChoiceLabel(useCaseChoice)} in {sdk}.
          </p>
        </div>
        <CodeHighlighter
          className={`${codeBlockClass} whitespace-pre-wrap`}
          code={generatedPrompt}
          language="text"
          maxHeight="360px"
          copy
        />
        <div className="flex flex-wrap items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            className={`w-fit ${focusRing}`}
            onClick={() => onPromptGenerated(useCaseChoice, sdk)}
          >
            Regenerate
          </Button>
        </div>
        <p className="text-sm text-muted-foreground">
          Your agent will scaffold the project, start a worker, and trigger a
          run. Come back here to watch it connect.
        </p>
      </>
    ),
    quickstart: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Scaffold a project</h3>
          <p className="text-sm text-muted-foreground">
            Generate a starter project and start a worker.
          </p>
        </div>
        {language === null ? (
          <p className="text-sm text-muted-foreground">
            Ruby does not have a scaffold template yet. Follow the{' '}
            <a
              href="https://docs.hatchet.run/llms/reference/ruby.md"
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:text-foreground"
            >
              Ruby SDK reference
            </a>{' '}
            to create a project, or switch to the agent setup.
          </p>
        ) : (
          <>
            <p className="text-sm">Scaffold the project:</p>
            <CodeHighlighter
              className={codeBlockClass}
              code={scaffoldCommand({ useCase, language })}
              language="shell"
              copy
            />
            <p className="text-sm">Start a dev worker:</p>
            <CodeHighlighter
              className={codeBlockClass}
              code={workerDevCommand(profileName)}
              language="shell"
              copy
            />
          </>
        )}
        <StatusRow
          ready={progress.workerConnected}
          done="Worker connected"
          waiting="Waiting for the worker to connect..."
        />
      </>
    ),
    runtask: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Run a task</h3>
          <p className="text-sm text-muted-foreground">
            Trigger the workflow and watch it complete.
          </p>
        </div>
        <p className="text-sm">Trigger a run:</p>
        <CodeHighlighter
          className={codeBlockClass}
          code={triggerCommand(useCase, profileName)}
          language="shell"
          copy
        />
        <StatusRow
          ready={progress.runCompleted}
          done="Run completed"
          waiting="Waiting for the run to complete..."
        />
      </>
    ),
    finish: (
      <>
        <div className="space-y-1">
          <h3 className="text-base font-semibold">Watch it connect</h3>
          <p className="text-sm text-muted-foreground">
            No need to refresh, we're watching things.
          </p>
        </div>
        <StatusRow
          ready={progress.workerConnected}
          done="Worker connected"
          waiting="Waiting for a worker to connect..."
        />
        <StatusRow
          ready={progress.runCompleted}
          done="Task run completed"
          waiting="Waiting for a task to run..."
        />
        {progress.onboarded && (
          <p className="text-sm font-medium text-green-500">
            You're all set up!
          </p>
        )}
        <Button
          variant="outline"
          size="default"
          className={continueButtonClass}
          disabled={!progress.onboarded}
          hoverText={
            progress.onboarded
              ? undefined
              : 'Waiting for a worker to connect and execute a task.'
          }
          onClick={onFinish}
        >
          Finish
          <CheckIcon className="size-3 text-brand" />
        </Button>
      </>
    ),
  };

  const currentIndex = sequence.indexOf(currentStep);
  const nextStep = sequence[currentIndex + 1];

  // Back steps within the sequence without confirming the selection or
  // emitting a step-change event, matching the prior Back behavior.
  const goBack = () => {
    const prev = sequence[currentIndex - 1];
    if (prev) {
      setCurrentStep(prev);
    }
  };

  // Next is the sole forward control. Moving off the MCP step regenerates the
  // prompt (and its analytics capture), preserving what the removed inline CTA
  // did.
  const goNext = () => {
    if (!nextStep) {
      return;
    }
    if (currentStep === 'mcp') {
      onPromptGenerated(useCaseChoice, sdk);
    }
    goTo(nextStep);
  };

  // The path selector advances by picking a card; finish keeps its own gated
  // CTA. Every other step advances via Next.
  const showNext =
    currentStep !== 'path' && currentStep !== 'finish' && Boolean(nextStep);

  return (
    <div className="space-y-6">
      <ol className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {sequence.map((step, index) => (
          <li key={step}>
            <button
              type="button"
              onClick={() => goTo(step)}
              className={cn(
                'font-medium transition-colors hover:text-foreground',
                focusRing,
                step === currentStep
                  ? 'text-foreground'
                  : index < currentIndex && 'text-foreground/70',
              )}
            >
              {index + 1}. {stepRailLabels[step]}
            </button>
          </li>
        ))}
      </ol>
      <div className="rounded-md px-6 py-6 bg-muted/20 ring-1 ring-border/50 ring-inset space-y-5">
        {stepContent[currentStep]}
        {/* Persistent, aligned footer: life-ring + Back on the left, Next on
            the right, all on one horizontal line. */}
        <div className="flex items-center justify-between border-t border-border/50 pt-4">
          <div className="flex items-center gap-2">
            <HelpDropdown
              icon={<LifeBuoy className="size-5 text-foreground" />}
              align="start"
              side="top"
              className="text-muted-foreground"
            />
            {currentIndex > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className={`gap-1 text-muted-foreground ${focusRing}`}
                onClick={goBack}
              >
                <ChevronLeftIcon className="size-3" />
                Back
              </Button>
            )}
            {path === 'manual' && currentStep !== 'finish' && (
              <Button
                variant="ghost"
                size="sm"
                className={`text-muted-foreground underline ${focusRing}`}
                onClick={() => choosePath('agent')}
              >
                Switch to the agent setup
              </Button>
            )}
          </div>
          {showNext && (
            <Button
              variant="default"
              size="sm"
              className={`gap-1 ${focusRing}`}
              onClick={goNext}
            >
              Next
              <ChevronRightIcon className="size-3" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
