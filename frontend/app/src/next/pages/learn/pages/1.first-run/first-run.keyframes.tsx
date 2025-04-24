import { Highlights } from '../../components/lesson-plan';
import { lessonPlan } from './first-run.lesson-plan';

export type HighlightFrames =
  | 'client'
  | 'task-name'
  | 'task-input'
  | 'task-fn'
  | 'worker'
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
    worker: {
      task: {
        lines: [1],
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
    run: {
      run: {
        lines: [1],
      },
    },
  },
};
