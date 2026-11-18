from collections.abc import Mapping
from typing import TypeVar

TLiteral = TypeVar("TLiteral", bound=str)
TConverted = TypeVar("TConverted")


def convert_literal_using_mapping(
    value: TLiteral,
    mapping: Mapping[TLiteral, TConverted],
    parameter_name: str,
) -> TConverted:
    if value not in mapping:
        raise ValueError(
            f"`{parameter_name}` must be one of {list(mapping)}. Got: {value!r}"
        )

    return mapping[value]
