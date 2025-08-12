import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# from hatchet_sdk import Hatchet\n# import pytest\n\n# from tests.utils import fixture_bg_worker\n\n\n# worker = fixture_bg_worker(["poetry", "run", "manual_trigger"])\n\n# # @pytest.mark.asyncio(loop_scope="session")\n# async def test_run(hatchet: Hatchet):\n#     # TODO\n',
  source: 'out/python/delayed/test_delayed.py',
  blocks: {},
  highlights: {},
};

export default snippet;
