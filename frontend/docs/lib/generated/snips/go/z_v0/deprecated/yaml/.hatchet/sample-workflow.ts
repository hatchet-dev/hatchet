import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "name: \"post-user-sign-up\"\nversion: v0.2.0\ntriggers:\n  events:\n    - user:create\njobs:\n  print-user:\n    steps:\n      - id: echo1\n        action: echo:echo\n        timeout: 60s\n        with:\n          message: \"Username is {{ .input.username }}\"\n      - id: echo2\n        action: echo:echo\n        timeout: 60s\n        with:\n          message: \"Above message is: {{ .steps.echo1.message }}\"\n      - id: echo3\n        action: echo:echo\n        timeout: 60s\n        with:\n          message: \"Above message is: {{ .steps.echo2.message }}\"\n      - id: testObject\n        action: echo:object\n        timeout: 60s\n        with:\n          object: \"{{ .steps.echo3.json }}\"\n",
  "source": "out/go/z_v0/deprecated/yaml/.hatchet/sample-workflow.yaml",
  "blocks": {},
  "highlights": {}
};

export default snippet;
