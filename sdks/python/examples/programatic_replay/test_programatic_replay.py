# from hatchet_sdk import Hatchet
# import pytest

# from tests.utils import fixture_bg_worker

# worker = fixture_bg_worker(["poetry", "run", "async"])

# # requires scope module or higher for shared event loop
# @pytest.mark.asyncio(scope="session")
# async def test_run(hatchet: Hatchet):
#     run = hatchet.admin.run_workflow("DagWorkflow", {})
#     result = await run.result()
#     assert result["step1"]["test"] == "test"
