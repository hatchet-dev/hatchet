import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    'name: "test-step-timeout"\nversion: v0.1.0\ntriggers:\n  events:\n    - user:create\njobs:\n  timeout-job:\n    steps:\n      - id: timeout\n        action: timeout:timeout\n        timeout: 5s\n      # This step should not be reached\n      - id: later-step\n        action: timeout:timeout\n        timeout: 5s\n',
  source: 'out/go/z_v0/deprecated/timeout/.hatchet/step-timeout-workflow.yaml',
  blocks: {},
  highlights: {},
};

export default snippet;
