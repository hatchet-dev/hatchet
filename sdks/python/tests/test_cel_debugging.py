from typing import Literal

import pytest

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_cel_debug_response_status import (
    V1CELDebugResponseStatus,
)
from hatchet_sdk.features.cel import CELSuccess  # noqa: F401 (used in unit test)
from hatchet_sdk.utils.typing import JSONSerializableMapping


@pytest.mark.parametrize(
    "expression, input, additional_metadata, filter_payload, expected_status, expected_output, expected_output_str, expected_output_int, expected_output_type",
    [
        (
            "input.key == 'value' && additional_metadata.meta == 'data' && payload.filter == 'payload'",
            {"key": "value"},
            {"meta": "data"},
            {"filter": "payload"},
            "success", True, None, None, "bool",
        ),
        (
            "input.key == 'value'",
            {"key": "other_value"},
            None, None,
            "success", False, None, None, "bool",
        ),
        (
            "input.key == 'value'",
            {}, None, None,
            "failure", None, None, None, None,
        ),
        (
            "input.user_id",
            {"user_id": "alice"},
            None, None,
            "success", None, "alice", None, "string",
        ),
        (
            "'singleton'",
            {}, None, None,
            "success", None, "singleton", None, "string",
        ),
        (
            "input.cost",
            {"cost": 5},
            None, None,
            "success", None, None, 5, "int",
        ),
        (
            "payload.tier == 'gold'",
            {}, None,
            {"tier": "gold"},
            "success", True, None, None, "bool",
        ),
        (
            "input.count",
            {"count": 0},
            None, None,
            "success", None, None, 0, "int",
        ),
        (
            "''",
            {}, None, None,
            "success", None, "", None, "string",
        ),
    ],
)
def test_cel_debug(
    hatchet: Hatchet,
    expression: str,
    input: JSONSerializableMapping,
    additional_metadata: JSONSerializableMapping | None,
    filter_payload: JSONSerializableMapping | None,
    expected_status: Literal["success", "failure"],
    expected_output: bool | None,
    expected_output_str: str | None,
    expected_output_int: int | None,
    expected_output_type: str | None,
) -> None:
    result = hatchet.cel.debug(
        expression=expression,
        input=input,
        additional_metadata=additional_metadata,
        filter_payload=filter_payload,
    )

    print(result)

    assert result.result.status == expected_status

    if result.result.status == "success":
        assert result.result.output == expected_output
        assert result.result.output_str == expected_output_str
        assert result.result.output_int == expected_output_int
        assert result.result.output_type == expected_output_type


def test_cel_success_wrong_type_helpers() -> None:
    string_result = CELSuccess(output_str="alice", output_type="string")
    with pytest.raises(ValueError):
        string_result.as_bool()
    with pytest.raises(ValueError):
        string_result.as_int()
    assert string_result.as_str() == "alice"

    bool_result = CELSuccess(output=True, output_type="bool")
    with pytest.raises(ValueError):
        bool_result.as_str()
    with pytest.raises(ValueError):
        bool_result.as_int()
    assert bool_result.as_bool() is True

    int_result = CELSuccess(output_int=5, output_type="int")
    with pytest.raises(ValueError):
        int_result.as_bool()
    with pytest.raises(ValueError):
        int_result.as_str()
    assert int_result.as_int() == 5
