import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import pytest\n\nfrom examples.logger.workflow import logging_workflow\n\n\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_run() -> None:\n    result = await logging_workflow.aio_run()\n\n    assert result[\"root_logger\"][\"status\"] == \"success\"\n",
  "source": "out/python/logger/test_logger.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
