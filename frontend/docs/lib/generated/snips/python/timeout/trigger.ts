import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from examples.timeout.worker import refresh_timeout_wf, timeout_wf\n\ntimeout_wf.run()\nrefresh_timeout_wf.run()\n",
  "source": "out/python/timeout/trigger.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
