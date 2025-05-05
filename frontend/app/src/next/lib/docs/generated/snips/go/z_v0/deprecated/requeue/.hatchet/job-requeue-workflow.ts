import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    "name: 'test-step-requeue'\nversion: v0.2.0\ntriggers:\n  events:\n    - example:event\njobs:\n  requeue-job:\n    steps:\n      - id: requeue\n        action: requeue:requeue\n        timeout: 10s\n",
  source: 'out/go/z_v0/deprecated/requeue/.hatchet/job-requeue-workflow.yaml',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
