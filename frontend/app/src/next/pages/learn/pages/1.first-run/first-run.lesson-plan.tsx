import {
  Card,
  CardTitle,
  CardDescription,
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

export type FirstRunStepKeys = 'setup' | 'task' | 'worker' | 'run';

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
      title: 'Setup Your Environment',
      description: SetupStep,
      githubCode: {
        typescript: 'src/hatchet-client.ts',
        python: 'src/hatchet_client.py',
        go: 'hatchet_client/hatchet_client.go',
      },
    },
    task: {
      title: 'Write Your First Task',
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
    <>
      <CardTitle>Learn Hatchet in 5 minutes</CardTitle>
      <CardDescription>
        Hatchet is a platform for building and running distributed applications.{' '}
        <br />
        <br />
        This lesson will walk you through the process of setting up your
        environment and running your first task.
      </CardDescription>
    </>
  );
}

function SetupStep() {
  const { language, setLanguage, extra, setExtra, commands } = useLesson();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [hasToken, setHasToken] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Setup Your Environment</CardTitle>
        <CardDescription>
          First, choose your technology stack and install the necessary
          dependencies
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 flex flex-col gap-4">
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
        Clone the Hatchet repository and navigate to the project directory and
        install the dependencies
        <Code
          title="cli"
          language={'bash'}
          value={`git clone https://github.com/${lessonPlan.codeBlockDefaults.repos[language]} && 
cd ${lessonPlan.codeBlockDefaults.repos[language].split('/').pop()} && 
${commands.install}
`}
        />
        Next, let's create an access token and save it to your project's `.env`
        file.
        <SignInRequiredAction
          description="Free tier users can follow along and run your code code locally."
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
        <Highlight frame="client">
          Great, this token is used by the hatchet client to connect to the
          Hatchet engine.
        </Highlight>
      </CardContent>
      {showTokenDialog && (
        <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
          <CreateTokenDialog
            close={() => setShowTokenDialog(false)}
            onSuccess={() => setHasToken(true)}
          />
        </Dialog>
      )}
    </Card>
  );
}

function TaskStep() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Write Your First Task</CardTitle>
        <CardDescription>
          In Hatchet, tasks are the fundamental unit of work. Let's create a
          simple task that transforms a message to lowercase.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>A task in Hatchet consists of:</p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <Highlight frame="task-name">A name to identify the task</Highlight>
          </li>
          <li>
            <Highlight frame="task-fn">
              A function that performs the actual work
            </Highlight>
          </li>
        </ul>
        <p>The task function receives two parameters:</p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <Highlight frame="task-input">
              input: The input data to the task
            </Highlight>
          </li>
          <li>
            <Highlight frame="task-ctx">
              ctx: A context object to interact with the task and hatchet from
              within a run
            </Highlight>
          </li>
        </ul>
        <p>
          In the code example to the right, we've created a simple task that:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <Highlight frame="task-name">Is named "simple"</Highlight>
          </li>
          <li>
            <Highlight frame="task-fn">
              Returns an object with a "transformed_message" field containing
              the lowercase version of the input message
            </Highlight>
          </li>
        </ul>
        <p>
          Once you've defined your task, you'll need to register it with a
          worker before you can run it. Let's move on to setting up a worker in
          the next step.
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
        <CardTitle>Register Your First Worker</CardTitle>
        <CardDescription>
          Workers are the backbone of Hatchet, responsible for executing your
          tasks. Let's set up a worker to run our simple task.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>
          Workers are long-running processes that execute your tasks. They
          communicate with the Hatchet engine to receive tasks and report
          results, enabling distributed execution across your infrastructure.
        </p>
        <p>
          In the code example to the right, we've{' '}
          <Highlight frame="worker">
            created a worker named "simple-worker"
          </Highlight>
          that registers our{' '}
          <Highlight frame="worker-task">simple task</Highlight>. The worker can
          process up to 100 tasks concurrently, giving it significant capacity
          to handle multiple requests.
        </p>
        <p>To start the worker, run:</p>
        <Code title="cli" language="bash" value={commands.startWorker} />
        <WorkerListener name="simple-worker" />
        <p>
          Once started, the worker will begin listening for tasks and continue
          running until stopped. You'll see logs indicating its status and
          readiness to process tasks.
        </p>
        <p>
          With both our task and worker in place, we're ready to run our first
          task!
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
        <CardTitle>Run Your Task</CardTitle>
        <CardDescription>
          Now that we have a task and a worker, let's run our first task!
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>
          Running a task in Hatchet is simple. You can use the `run` method on
          your task instance, passing in the required input. The task will be
          executed by an available worker, and you'll receive the result once
          it's complete.
        </p>
        <p>
          In the code example to the right, we're running our "simple" task with
          a message input. The task will transform the message to lowercase and
          return the result.
        </p>

        <div>
          <p className="mb-2">Run from CLI:</p>
          <Code title="cli" language="bash" value={commands.runTask} />
        </div>
        <TaskExecution
          name="first-workflow"
          input={{ message: 'Hello, World!' }}
          onRun={(link) => {
            setRunLink(link);
          }}
        />
        <p>
          Congratulations! You've successfully set up and run your first Hatchet
          task. This is just the beginning - Hatchet offers many more features
          for building complex workflows and managing task execution.
        </p>
        {runLink && (
          <div className="flex justify-end">
            <Link to={runLink}>
              <Button>
                View run details <ArrowUpRight className="h-4 w-4" />
              </Button>
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
