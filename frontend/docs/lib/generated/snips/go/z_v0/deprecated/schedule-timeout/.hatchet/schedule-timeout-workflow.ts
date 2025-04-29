import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'unknown',
  'content': 'name: \'test-schedule-timeout\'\nversion: v0.1.0\ntriggers:\n  events:\n    - user:create\njobs:\n  timeout-job:\n    steps:\n      - id: timeout\n        action: timeout:timeout\n',
  'source': 'out/go/z_v0/deprecated/schedule-timeout/.hatchet/schedule-timeout-workflow.yaml',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
