from subprocess import Popen
from typing import Any
from uuid import uuid4

import grpc
import pytest

from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet, TriggerWorkflowOptions
from tests.engine_null_unicode_character.task import (
    Message,
    engine_null_unicode_rejection,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [(["poetry", "run", "python", "tests/worker.py"], 8099)],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_unicode_rejection(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    test_run_id = str(uuid4())

    with pytest.raises(
        grpc.RpcError,
        match=r"encoded jsonb contains invalid null character .* in field `payload`",
    ):
        await engine_null_unicode_rejection.aio_run(
            input=Message(content="Hello\x00World"),
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id}
            ),
        )

    with pytest.raises(
        FailedTaskRunExceptionGroup,
        match=r"encoded jsonb contains invalid null character .* in field `taskOutput`",
    ):
        await engine_null_unicode_rejection.aio_run(
            input=Message(content="Hello World"),
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id}
            ),
        )
