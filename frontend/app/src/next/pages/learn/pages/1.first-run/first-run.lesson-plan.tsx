import {
  Card,
  CardTitle,
  CardDescription,
  CardHeader,
  CardContent,
} from '@/next/components/ui/card';
import { LessonPlan, LessonPlanStepProps } from '../../components/lesson-plan';
import { Tabs, TabsTrigger } from '@/next/components/ui/tabs';
import { TabsList } from '@/next/components/ui/tabs';

export type FirstRunStepKeys = 'setup' | 'task' | 'worker' | 'run';

export type FirstRunExtra = {
  packageManager: 'npm' | 'pnpm' | 'yarn';
};

export type Props = LessonPlanStepProps<FirstRunStepKeys, FirstRunExtra>;

export const lessonPlan: LessonPlan<FirstRunStepKeys, FirstRunExtra> = {
  title: 'First Run',
  description: 'Learn how to get started with Hatchet',
  extraDefaults: {
    packageManager: 'npm',
  },
  codeBlockDefaults: {
    repo: 'hatchet-dev/hatchet-typescript-quickstart',
    language: 'typescript',
    showLineNumbers: false,
  },
  steps: {
    setup: {
      title: 'Setup Your Environment',
      description: SetupStep,
      githubCode: {
        path: 'src/hatchet-client.ts',
      },
    },
    task: {
      title: 'Write Your First Task',
      description: TaskStep,
      githubCode: {
        path: 'src/workflows/first-workflow.ts',
      },
    },
    worker: {
      title: 'Register Your First Worker',
      description: WorkerStep,
      githubCode: {
        path: 'src/worker.ts',
      },
    },
    run: {
      title: 'Run Your Task',
      description: RunStep,
      githubCode: {
        path: 'src/run.ts',
      },
    },
  },
};

function SetupStep({ setHighlights, setFocus, extra, setExtra }: Props) {
  return (
    <Card
      onMouseEnter={() =>
        setHighlights('setup', {
          lines: [1],
          strings: ['Hatchet'],
        })
      }
      onMouseLeave={() => setHighlights('setup')}
      onClick={() => setFocus('setup')}
    >
      <CardHeader>
        <CardTitle>Setup Your Environment</CardTitle>
        <CardDescription>
          Choose your technology stack and install the necessary dependencies
          {extra.packageManager}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs
          value={extra.packageManager}
          onValueChange={(value) => setExtra({ packageManager: value as any })}
        >
          <TabsList>
            <TabsTrigger value="npm">npm</TabsTrigger>
            <TabsTrigger value="pnpm">pnpm</TabsTrigger>
            <TabsTrigger value="yarn">yarn</TabsTrigger>
          </TabsList>
        </Tabs>
      </CardContent>
    </Card>
  );
}

function TaskStep({ setHighlights }: Props) {
  return (
    <Card
      onMouseEnter={() =>
        setHighlights('setup', {
          lines: [1],
          strings: ['Hatchet'],
        })
      }
      onMouseLeave={() => setHighlights('setup')}
    >
      <CardHeader>
        <CardTitle>Write Your First Task</CardTitle>
        <CardDescription>
          Choose your technology stack and install the necessary dependencies
        </CardDescription>
      </CardHeader>
    </Card>
  );
}

function WorkerStep({ setHighlights }: Props) {
  return (
    <Card
      onMouseEnter={() =>
        setHighlights('setup', {
          lines: [1],
          strings: ['Hatchet'],
        })
      }
      onMouseLeave={() => setHighlights('setup')}
    >
      <CardHeader>
        <CardTitle>Register Your First Worker</CardTitle>
        <CardDescription>
          Choose your technology stack and install the necessary dependencies
        </CardDescription>
      </CardHeader>
    </Card>
  );
}

function RunStep({ setHighlights }: Props) {
  return (
    <Card
      onMouseEnter={() =>
        setHighlights('setup', {
          lines: [1],
          strings: ['Hatchet'],
        })
      }
      onMouseLeave={() => setHighlights('setup')}
    >
      <CardHeader>
        <CardTitle>Setup Your Environment</CardTitle>
        <CardDescription>
          Choose your technology stack and install the necessary dependencies
        </CardDescription>
      </CardHeader>
    </Card>
  );
}
