import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\n\nfrom examples.streaming.worker import streaming_workflow\n\n\ndef main() -> None:\n    ref = streaming_workflow.run_no_wait()\n    time.sleep(1)\n\n    stream = ref.stream()\n\n    for chunk in stream:\n        print(chunk)\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/streaming/sync_stream.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
