import { Highlights } from '@/next/learn/components';
import { lessonPlan } from './first-run.lesson-plan';

export type HighlightFrames =
  | 'client'
  | 'task-name'
  | 'task-input'
  | 'task-ctx'
  | 'task-fn'
  | 'worker'
  | 'worker-task'
  | 'run';

export const codeKeyFrames: Highlights<
  HighlightFrames,
  keyof typeof lessonPlan.steps
> = {
  typescript: {
    client: {
      setup: {
        lines: [3],
      },
    },
    'task-name': {
      task: {
        lines: [9],
      },
    },
    'task-input': {
      task: {
        lines: [10],
        strings: ['input'],
      },
    },
    'task-ctx': {
      task: {
        lines: [10],
        strings: ['ctx'],
      },
    },
    'task-fn': {
      task: {
        lines: [10, 11, 12, 13],
      },
    },
    worker: {
      worker: {
        lines: [5, 6, 7],
        strings: ['simple-worker'],
      },
    },
    'worker-task': {
      worker: {
        lines: [5, 6, 7],
        strings: ['simple'],
      },
    },
    run: {
      run: {
        lines: [1],
      },
    },
  },
  python: {
    client: {
      setup: {
        lines: [4],
      },
    },
    worker: {
      task: {
        lines: [1],
      },
    },
    'task-name': {
      task: {
        lines: [1],
      },
    },
    'task-input': {
      task: {
        lines: [1],
        strings: ['input'],
      },
    },
    'task-ctx': {
      task: {
        lines: [11],
        strings: ['ctx'],
      },
    },
    'task-fn': {
      task: {
        lines: [10, 11, 12, 13],
      },
    },
    run: {
      run: {
        lines: [1],
      },
    },
    'worker-task': {},
  },
  go: {
    client: {
      setup: {
        lines: [4],
      },
    },
    worker: {
      task: {
        lines: [1],
      },
    },
    'task-name': {
      task: {
        lines: [1],
      },
    },
    'task-input': {
      task: {
        lines: [1],
      },
    },
    'task-fn': {
      task: {
        lines: [1],
      },
    },
    'task-ctx': {
      task: {
        lines: [1],
      },
    },
    run: {
      run: {
        lines: [1],
      },
    },
    'worker-task': {},
  },
};
