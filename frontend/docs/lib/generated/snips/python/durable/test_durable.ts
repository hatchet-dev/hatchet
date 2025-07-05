import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nimport pytest\n\nfrom examples.durable.worker import EVENT_KEY, SLEEP_TIME, durable_workflow\nfrom hatchet_sdk import Hatchet\n\n\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_durable(hatchet: Hatchet) -> None:\n    ref = durable_workflow.run_no_wait()\n\n    await asyncio.sleep(SLEEP_TIME + 10)\n\n    hatchet.event.push(EVENT_KEY, {\"test\": \"test\"})\n\n    result = await ref.aio_result()\n\n    workers = await hatchet.workers.aio_list()\n\n    assert workers.rows\n\n    active_workers = [w for w in workers.rows if w.status == \"ACTIVE\"]\n\n    assert len(active_workers) == 2\n    assert any(\n        w.name == hatchet.config.apply_namespace(\"e2e-test-worker\")\n        for w in active_workers\n    )\n    assert any(\n        w.name == hatchet.config.apply_namespace(\"e2e-test-worker_durable\")\n        for w in active_workers\n    )\n\n    assert result[\"durable_task\"][\"status\"] == \"success\"\n\n    wait_group_1 = result[\"wait_for_or_group_1\"]\n    wait_group_2 = result[\"wait_for_or_group_2\"]\n\n    assert abs(wait_group_1[\"runtime\"] - SLEEP_TIME) < 3\n\n    assert wait_group_1[\"key\"] == wait_group_2[\"key\"]\n    assert wait_group_1[\"key\"] == \"CREATE\"\n    assert \"sleep\" in wait_group_1[\"event_id\"]\n    assert \"event\" in wait_group_2[\"event_id\"]\n",
  "source": "out/python/durable/test_durable.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
