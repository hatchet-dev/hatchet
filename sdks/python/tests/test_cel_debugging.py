from typing import Literal

import pytest

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_cel_debug_response_status import \
    V1CELDebugResponseStatus
from hatchet_sdk.utils.typing import JSONSerializableMapping


@pytest.mark.parametrize(
    "expression, input, additional_metadata, filter_payload, expected",
    [
        (
            "input.key == 'value' && additional_metadata.meta == 'data' && payload.filter == 'payload'",
            {"key": "value"},
            {"meta": "data"},
            {"filter": "payload"},
            ("success", True),
        ),
        (
            "input.key == 'value'",
            {"key": "other_value"},
            None,
            None,
            ("success", False),
        ),
        (
            "input.key == 'value'",
            {},
            None,
            None,
            ("failure", False),
        ),
    ],
)
def test_cel_debug(
    hatchet: Hatchet,
    expression: str,
    input: JSONSerializableMapping,
    additional_metadata: JSONSerializableMapping | None,
    filter_payload: JSONSerializableMapping | None,
    expected: tuple[Literal["success", "failure"], bool],  ## (error, result)
) -> None:
    result = hatchet.cel.debug(
        expression=expression,
        input=input,
        additional_metadata=additional_metadata,
        filter_payload=filter_payload,
    )

    status, output = expected

    print(result)

    assert result.result.status == status

    if result.result.status == "success":
        assert result.result.output == output
