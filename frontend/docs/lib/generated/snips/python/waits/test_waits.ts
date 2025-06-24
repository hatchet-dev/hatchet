import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nimport pytest\n\nfrom examples.waits.worker import task_condition_workflow\nfrom hatchet_sdk import Hatchet\n\n\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_waits(hatchet: Hatchet) -> None:\n\n    ref = task_condition_workflow.run_no_wait()\n\n    await asyncio.sleep(15)\n\n    hatchet.event.push(\"skip_on_event:skip\", {})\n    hatchet.event.push(\"wait_for_event:start\", {})\n\n    result = await ref.aio_result()\n\n    assert result[\"skip_on_event\"] == {\"skipped\": True}\n\n    first_random_number = result[\"start\"][\"random_number\"]\n    wait_for_event_random_number = result[\"wait_for_event\"][\"random_number\"]\n    wait_for_sleep_random_number = result[\"wait_for_sleep\"][\"random_number\"]\n\n    left_branch = result[\"left_branch\"]\n    right_branch = result[\"right_branch\"]\n\n    assert left_branch.get(\"skipped\") is True or right_branch.get(\"skipped\") is True\n\n    branch_random_number = left_branch.get(\"random_number\") or right_branch.get(\n        \"random_number\"\n    )\n\n    result_sum = result[\"sum\"][\"sum\"]\n\n    assert (\n        result_sum\n        == first_random_number\n        + wait_for_event_random_number\n        + wait_for_sleep_random_number\n        + branch_random_number\n    )\n",
  "source": "out/python/waits/test_waits.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
