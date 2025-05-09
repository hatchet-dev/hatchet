import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'unknown',
  'content': 'name: \'test-job-timeout\'\nversion: v0.1.0\ntriggers:\n  events:\n    - user:create\njobs:\n  timeout-job:\n    timeout: 3s\n    steps:\n      - id: timeout\n        action: timeout:timeout\n        timeout: 10s\n',
  'source': 'out/go/z_v0/deprecated/timeout/.hatchet/job-timeout-workflow.yaml',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
