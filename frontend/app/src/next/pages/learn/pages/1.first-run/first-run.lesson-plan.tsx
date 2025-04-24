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
import { useEffect, useState } from 'react';
import { CheckCircle2, Loader2 } from 'lucide-react';
import { codeKeyFrames } from './first-run.keyframes';
import { Highlight } from '@/next/learn/components';
import { useLesson as untypedUseLesson } from '@/next/learn/hooks/use-lesson';

const useLesson = untypedUseLesson<
  FirstRunStepKeys,
  FirstRunExtra,
  CommandConfig
>;

export type FirstRunStepKeys = 'intro' | 'setup' | 'task' | 'worker' | 'run';

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
  steps: {
    intro: {
      title: 'Welcome to Hatchet',
      description: IntroStep,
    },
    setup: {
      title: 'Setup Your Environment',
      description: SetupStep,
      githubCode: {
        typescript: 'src/hatchet-client.ts',
        python: 'src/hatchet_client.py',
        go: 'src/hatchet_client.go',
      },
    },
    task: {
      title: 'Write Your First Task',
      description: TaskStep,
      githubCode: {
        typescript: 'src/workflows/first-workflow.ts',
        python: 'src/workflows/first_workflow.py',
        go: 'src/workflows/first_workflow.go',
      },
    },
    worker: {
      title: 'Register Your First Worker',
      description: WorkerStep,
      githubCode: {
        typescript: 'src/worker.ts',
        python: 'src/worker.py',
        go: 'src/worker.go',
      },
    },
    run: {
      title: 'Run Your Task',
      description: RunStep,
      githubCode: {
        typescript: 'src/run.ts',
        python: 'src/run.py',
        go: 'src/run.go',
      },
    },
  },
};

function IntroStep() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Welcome to Hatchet</CardTitle>
        <CardDescription>
          Hatchet is a platform for building and running distributed
          applications. <br />
          <br />
          This lesson will walk you through the process of setting up your
          environment and running your first task.
        </CardDescription>
      </CardHeader>
    </Card>
  );
}

function SetupStep() {
  const { language, setLanguage, extra, setExtra, setHighlights, commands } =
    useLesson();

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
        <Button>Create Access Token</Button>
        <Highlight
          frame="client"
          language={language}
          codeKeyFrames={codeKeyFrames}
          setHighlights={setHighlights}
        >
          Great, this token is used by the hatchet client to connect to the
          Hatchet engine.
        </Highlight>
      </CardContent>
    </Card>
  );
}

function TaskStep() {
  const { language, setHighlights } = useLesson();

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
            <Highlight
              frame="task-name"
              language={language}
              codeKeyFrames={codeKeyFrames}
              setHighlights={setHighlights}
            >
              A name to identify the task
            </Highlight>
          </li>
          <li>
            <Highlight
              frame="task-input"
              language={language}
              codeKeyFrames={codeKeyFrames}
              setHighlights={setHighlights}
            >
              An input validator to define the expected input structure
            </Highlight>
          </li>
          <li>
            <Highlight
              frame="task-fn"
              language={language}
              codeKeyFrames={codeKeyFrames}
              setHighlights={setHighlights}
            >
              A function that performs the actual work
            </Highlight>
          </li>
        </ul>
        <p>The task function receives two parameters:</p>
        <ul className="list-disc pl-6 space-y-2">
          <li>input: The validated input data</li>
          <li>
            ctx: A context object containing metadata about the task execution
          </li>
        </ul>
        <p>
          In the code example to the right, we've created a simple task that:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>Is named "simple"</li>
          <li>Accepts an input with a "message" string field</li>
          <li>
            Returns an object with a "transformed_message" field containing the
            lowercase version of the input message
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

function WorkerConnection() {
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLoading(false);
      setIsConnected(true);
    }, 3000);

    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="flex items-center gap-2 p-4 bg-muted rounded-lg">
      {isLoading ? (
        <>
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Waiting for worker to connect...</span>
        </>
      ) : (
        <>
          <CheckCircle2 className="h-4 w-4 text-green-500" />
          <span>Worker connected successfully!</span>
        </>
      )}
    </div>
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
          In the code example to the right, we've created a worker named
          "simple-worker" that's configured to handle our "simple" task. The
          worker can process up to 100 tasks concurrently, giving it significant
          capacity to handle multiple requests.
        </p>
        <p>To start the worker, run:</p>
        <Code title="cli" language="bash" value={commands.startWorker} />
        <WorkerConnection />
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

function TaskExecution() {
  const { commands } = useLesson();
  const [isExecuting, setIsExecuting] = useState(false);
  const [isComplete, setIsComplete] = useState(false);
  const [result, setResult] = useState<string | null>(null);

  useEffect(() => {
    if (isExecuting) {
      const timer = setTimeout(() => {
        setIsExecuting(false);
        setIsComplete(true);
        setResult('hello world');
      }, 2000);

      return () => clearTimeout(timer);
    }
  }, [isExecuting]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        <div>
          <p className="mb-2">Run from CLI:</p>
          <Code title="cli" language="bash" value={commands.runTask} />
        </div>
        <div className="flex items-center gap-2 p-4 bg-muted rounded-lg">
          {!isExecuting && !isComplete && (
            <Button onClick={() => setIsExecuting(true)}>Run Task</Button>
          )}
          {isExecuting && (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>Executing task...</span>
            </>
          )}
          {isComplete && (
            <>
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span>Task completed successfully!</span>
            </>
          )}
        </div>
      </div>
      {result && (
        <div className="p-4 bg-muted rounded-lg">
          <p className="font-mono">{result}</p>
        </div>
      )}
    </div>
  );
}

function RunStep() {
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
        <TaskExecution />
        <p>
          Congratulations! You've successfully set up and run your first Hatchet
          task. This is just the beginning - Hatchet offers many more features
          for building complex workflows and managing task execution.
        </p>
      </CardContent>
    </Card>
  );
}
