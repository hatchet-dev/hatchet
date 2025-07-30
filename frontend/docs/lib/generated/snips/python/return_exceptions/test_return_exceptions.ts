import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import asyncio\n\nimport pytest\n\nfrom examples.return_exceptions.worker import Input, return_exceptions_task\n\n\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_return_exceptions_async() -> None:\n    results = await return_exceptions_task.aio_run_many(\n        [\n            return_exceptions_task.create_bulk_run_item(input=Input(index=i))\n            for i in range(10)\n        ],\n        return_exceptions=True,\n    )\n\n    for i, result in enumerate(results):\n        if i % 2 == 0:\n            assert isinstance(result, Exception)\n            assert f\"error in task with index {i}\" in str(result)\n        else:\n            assert result == {\"message\": \"this is a successful task.\"}\n\n\ndef test_return_exceptions_sync() -> None:\n    results = return_exceptions_task.run_many(\n        [\n            return_exceptions_task.create_bulk_run_item(input=Input(index=i))\n            for i in range(10)\n        ],\n        return_exceptions=True,\n    )\n\n    for i, result in enumerate(results):\n        if i % 2 == 0:\n            assert isinstance(result, Exception)\n            assert f\"error in task with index {i}\" in str(result)\n        else:\n            assert result == {\"message\": \"this is a successful task.\"}\n",
  "source": "out/python/return_exceptions/test_return_exceptions.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
