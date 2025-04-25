import {
  Card,
  CardTitle,
  CardHeader,
  CardContent,
} from '@/next/components/ui/card';
import {
  LessonPlan,
  SupportedLanguage,
  PackageManager,
} from '@/next/learn/components/lesson-plan';
import { Tabs, TabsTrigger } from '@/next/components/ui/tabs';
import { TabsList } from '@/next/components/ui/tabs';
import { CommandConfig, commands } from './first-runs.commands';
import { Code } from '@/next/components/ui/code';
import { Button } from '@/next/components/ui/button';
import { useState } from 'react';
import { ArrowUpRight, CheckCircle2, Key } from 'lucide-react';
import { codeKeyFrames } from './first-run.keyframes';
import { Highlight } from '@/next/learn/components';
import { useLesson as untypedUseLesson } from '@/next/learn/hooks/use-lesson';
import { SignInRequiredAction } from '@/next/pages/learn/components/signin-required-action';
import { Dialog } from '@/next/components/ui/dialog';
import { CreateTokenDialog } from '@/next/pages/authenticated/dashboard/settings/api-tokens/components/create-token-dialog';
import { TaskExecution } from '@/next/pages/learn/components/task-executor';
import { WorkerListener } from '@/next/pages/learn/components/worker-listener';
import { Link } from 'react-router-dom';

const useLesson = untypedUseLesson<
  FirstRunStepKeys,
  FirstRunExtra,
  CommandConfig
>;

export type FirstRunStepKeys = 'setup' | 'client' | 'task' | 'worker' | 'run';

export type FirstRunExtra = {
  language: SupportedLanguage;
  packageManager: PackageManager;
};

export const lessonPlan: LessonPlan<
  FirstRunStepKeys,
  FirstRunExtra,
  CommandConfig
> = {
  title: 'First Run',
  description: 'Learn how to get started with Hatchet',
  defaultLanguage: 'typescript',
  commands,
  codeKeyFrames,
  extraDefaults: {
    typescript: {
      packageManager: 'npm',
    },
    python: {
      packageManager: 'poetry',
    },
    go: {
      packageManager: 'go',
    },
  },
  codeBlockDefaults: {
    showLineNumbers: false,
    repos: {
      typescript: 'hatchet-dev/hatchet-typescript-quickstart',
      python: 'hatchet-dev/hatchet-python-quickstart',
      go: 'hatchet-dev/hatchet-go-quickstart',
    },
  },
  intro: <IntroStep />,
  duration: '~5 minutes',
  steps: {
    setup: {
      title: 'Environment Setup',
      description: SetupStep,
    },
    client: {
      title: 'Configure Hatchet Client',
      description: ClientStep,
      githubCode: {
        typescript: 'src/hatchet-client.ts',
        python: 'src/hatchet_client.py',
        go: 'hatchet_client/hatchet_client.go',
      },
    },
    task: {
      title: 'Define a Task',
      description: TaskStep,
      githubCode: {
        typescript: 'src/workflows/first-workflow.ts',
        python: 'src/workflows/first_workflow.py',
        go: 'workflows/first_workflow.go',
      },
    },
    worker: {
      title: 'Register Your First Worker',
      description: WorkerStep,
      githubCode: {
        typescript: 'src/worker.ts',
        python: 'src/worker.py',
        go: 'cmd/worker/main.go',
      },
    },
    run: {
      title: 'Run Your Task',
      description: RunStep,
      githubCode: {
        typescript: 'src/run.ts',
        python: 'src/run.py',
        go: 'cmd/run/main.go',
      },
    },
  },
};

function IntroStep() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-center">
      <div className="order-2 md:order-1 space-y-4">
        <h2 className="text-2xl font-bold">
          <span className="mr-2">🪓</span> Getting Started with Hatchet
        </h2>
        <p className="text-base text-muted-foreground">
          Hatchet provides a lightweight platform for orchestrating distributed
          tasks with minimal setup overhead.
        </p>
        <p className="text-base text-muted-foreground font-medium">
          This quick tutorial will guide you through the basics:
        </p>
        <ul className="list-disc pl-6 space-y-2 text-base text-muted-foreground">
          <li>Setting up your local environment</li>
          <li>Defining and registering task functions</li>
          <li>Starting a worker process to execute tasks</li>
        </ul>
        <p className="text-base text-muted-foreground font-medium">
          Estimated time: 5 minutes
        </p>
      </div>
      <div className="order-1 md:order-2 flex items-center justify-center bg-gray-100 rounded-lg w-full aspect-video">
        <span className="text-muted-foreground">[Graphic Placeholder]</span>
      </div>
    </div>
  );
}

function SetupStep() {
  const { language, setLanguage, extra, setExtra, commands } = useLesson();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [hasToken, setHasToken] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle>1. Setup Your Environment</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 flex flex-col gap-4">
        <p>Let's start by setting up your environment for local development.</p>
        <p>First, select your preferred language and package manager.</p>
        <Tabs
          value={language}
          onValueChange={(value) => setLanguage(value as SupportedLanguage)}
        >
          <TabsList>
            <TabsTrigger value="typescript">TypeScript</TabsTrigger>
            <TabsTrigger value="python">Python</TabsTrigger>
            <TabsTrigger value="go">Go</TabsTrigger>
          </TabsList>
        </Tabs>
        {language === 'typescript' && (
          <Tabs
            value={extra.packageManager}
            onValueChange={(value) =>
              setExtra({
                packageManager: value as PackageManager,
              })
            }
          >
            <TabsList>
              <TabsTrigger value="npm">npm</TabsTrigger>
              <TabsTrigger value="pnpm">pnpm</TabsTrigger>
              <TabsTrigger value="yarn">yarn</TabsTrigger>
            </TabsList>
          </Tabs>
        )}
        {language === 'python' && (
          <Tabs
            value={extra.packageManager}
            onValueChange={(value) =>
              setExtra({
                packageManager: value as PackageManager,
              })
            }
          >
            <TabsList>
              <TabsTrigger value="poetry">poetry</TabsTrigger>
              <TabsTrigger value="pip">pip</TabsTrigger>
              <TabsTrigger value="pipenv">pipenv</TabsTrigger>
            </TabsList>
          </Tabs>
        )}
        <p>Next, clone the starter repository and install dependencies:</p>
        <Code
          title="cli"
          language={'bash'}
          value={`git clone https://github.com/${lessonPlan.codeBlockDefaults.repos[language]} &&
cd ${lessonPlan.codeBlockDefaults.repos[language].split('/').pop()} &&
${commands.install}
`}
        />
        <p>
          Last setup step is to create an API token to authenticate with Hatchet
          and add it to the project's .env file.
        </p>
        <SignInRequiredAction
          description="API tokens enable your client to connect to Hatchet. Free tier users can follow along and run code locally."
          variant="card"
        >
          {!hasToken ? (
            <Button onClick={() => setShowTokenDialog(true)}>
              <Key className="h-4 w-4" />
              Create Access Token
            </Button>
          ) : (
            <Button
              variant="secondary"
              onClick={() => setShowTokenDialog(true)}
            >
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              Token Created
            </Button>
          )}
        </SignInRequiredAction>
      </CardContent>
      {showTokenDialog && (
        <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
          <CreateTokenDialog
            close={() => setShowTokenDialog(false)}
            onSuccess={() => setHasToken(true)}
          />
        </Dialog>
      )}
      <p>Now we can get into the code.</p>
    </Card>
  );
}

function ClientStep() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>2. Configure the Hatchet Client</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>
          The Hatchet client provides a programmatic interface to the Hatchet
          API, allowing you to define and execute tasks.
        </p>
        <p>
          It is recommended to instantiate the client in a separate file from
          your main application code and import it as needed.
        </p>
      </CardContent>
    </Card>
  );
}

function TaskStep() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>3. Declare a Simple Task</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>The most basic Hatchet task contains two key components:</p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <Highlight frame="task-name">A unique identifier (name)</Highlight>
          </li>
          <li>
            <Highlight frame="task-fn">An implementation function</Highlight>
          </li>
        </ul>
        <p>Task functions accept two parameters:</p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <Highlight frame="task-input">
              input: Strongly-typed input data
            </Highlight>
          </li>
          <li>
            <Highlight frame="task-ctx">
              ctx: Context object for task metadata and utilities
            </Highlight>
          </li>
        </ul>
        <p>
          As your service grows, you can add more control over task execution
          like rate limiting, retry logic, concurrency control, and more.
        </p>
        <p>
          Next, we'll register this task with a worker so it can be executed.
        </p>
      </CardContent>
    </Card>
  );
}

function WorkerStep() {
  const { commands } = useLesson();

  return (
    <Card>
      <CardHeader>
        <CardTitle>4. Start a Local Worker Process</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>Workers are long lived processes that execute your tasks.</p>
        <p>
          The sample code{' '}
          <Highlight frame="worker">
            initializes a worker named "simple-worker"
          </Highlight>{' '}
          that registers our{' '}
          <Highlight frame="worker-task">"simple" task</Highlight> with a
          <Highlight frame="worker-slots">max 100 concurrent runs</Highlight> on
          this worker instance.
        </p>
        <p>
          You can run multiple workers to execute tasks in parallel, and scale
          based on the number of tasks in your queue.
        </p>
        <p>In a new terminal, start the worker with:</p>
        <Code title="cli" language="bash" value={commands.startWorker} />
        <WorkerListener name="simple-worker" />
        <p>
          The worker process will remain active and ready to process tasks until
          it is terminated.
        </p>
      </CardContent>
    </Card>
  );
}

function RunStep() {
  const { commands } = useLesson();
  const [runLink, setRunLink] = useState<string | null>(null);
  return (
    <Card>
      <CardHeader>
        <CardTitle>5. Execute Tasks</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>
          With your task defined and worker running, you can now trigger task
          execution. The request lifecycle looks like this:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            Task execution requests are sent to the Hatchet engine via the API
          </li>
          <li>The engine routes the task to an available worker to run</li>
          <li>Results are stored and returned to the client</li>
        </ul>
        <p>
          The sample code executes our "simple" task with a "Hello, World!"
          message input and awaits the result.
        </p>

        <div>
          <p className="mb-2">
            Execute from CLI (open a separate terminal from your worker):
          </p>
          <Code title="cli" language="bash" value={commands.runTask} />
        </div>

        <p>Or, execute from the UI:</p>
        <TaskExecution
          name="first-workflow"
          input={{ message: 'Hello, World!' }}
          onRun={(link) => {
            setRunLink(link);
          }}
        />
        {runLink && (
          <>
            <p>
              Congratulations! You've successfully set up a Hatchet task and
              executed it through a distributed worker architecture.
            </p>
            <div className="flex justify-end">
              <Link to={runLink}>
                <Button>
                  View execution details <ArrowUpRight className="h-4 w-4" />
                </Button>
              </Link>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
