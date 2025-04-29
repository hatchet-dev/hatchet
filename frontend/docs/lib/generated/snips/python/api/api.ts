import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\ndef main() -> None:\n    workflow_list = hatchet.workflows.list()\n    rows = workflow_list.rows or []\n\n    for workflow in rows:\n        print(workflow.name)\n        print(workflow.metadata.id)\n        print(workflow.metadata.created_at)\n        print(workflow.metadata.updated_at)\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/api/api.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
