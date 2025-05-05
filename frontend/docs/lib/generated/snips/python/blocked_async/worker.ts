import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import hashlib\nimport time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# WARNING: this is an example of what NOT to do\n# This workflow is intentionally blocking the main thread\n# and will block the worker from processing other workflows\n#\n# You do not want to run long sync functions in an async def function\n\nblocked_worker_workflow = hatchet.workflow(name=\'Blocked\')\n\n\n@blocked_worker_workflow.task(execution_timeout=timedelta(seconds=11), retries=3)\nasync def step1(input: EmptyModel, ctx: Context) -> dict[str, str | int | float]:\n    print(\'Executing step1\')\n\n    # CPU-bound task: Calculate a large number of SHA-256 hashes\n    start_time = time.time()\n    iterations = 10_000_000\n    for i in range(iterations):\n        hashlib.sha256(f\'data{i}\'.encode()).hexdigest()\n\n    end_time = time.time()\n    execution_time = end_time - start_time\n\n    print(f\'Completed {iterations} hash calculations in {execution_time:.2f} seconds\')\n\n    return {\n        \'step1\': \'step1\',\n        \'iterations\': iterations,\n        \'execution_time\': execution_time,\n    }\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'blocked-worker\', slots=3, workflows=[blocked_worker_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/blocked_async/worker.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
