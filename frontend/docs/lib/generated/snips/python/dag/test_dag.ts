import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import pytest\n\nfrom examples.dag.worker import dag_workflow\nfrom hatchet_sdk import Hatchet\n\n\n@pytest.mark.asyncio(loop_scope=\'session\')\nasync def test_run(hatchet: Hatchet) -> None:\n    result = await dag_workflow.aio_run()\n\n    one = result[\'step1\'][\'random_number\']\n    two = result[\'step2\'][\'random_number\']\n    assert result[\'step3\'][\'sum\'] == one + two\n    assert result[\'step4\'][\'step4\'] == \'step4\'\n',
  'source': 'out/python/dag/test_dag.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
