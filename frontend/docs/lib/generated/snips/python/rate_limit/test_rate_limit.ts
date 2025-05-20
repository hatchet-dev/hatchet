import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\nimport time\n\nimport pytest\n\nfrom examples.rate_limit.worker import rate_limit_workflow\n\n\n@pytest.mark.skip(reason=\"The timing for this test is not reliable\")\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_run() -> None:\n\n    run1 = rate_limit_workflow.run_no_wait()\n    run2 = rate_limit_workflow.run_no_wait()\n    run3 = rate_limit_workflow.run_no_wait()\n\n    start_time = time.time()\n\n    await asyncio.gather(run1.aio_result(), run2.aio_result(), run3.aio_result())\n\n    end_time = time.time()\n\n    total_time = end_time - start_time\n\n    assert (\n        1 <= total_time <= 5\n    ), f\"Expected runtime to be a bit more than 1 seconds, but it took {total_time:.2f} seconds\"\n",
  "source": "out/python/rate_limit/test_rate_limit.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
