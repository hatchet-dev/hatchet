from typing import Any, get_args

import pytest

from hatchet_sdk.contracts.workflows_pb2 import (
    RateLimitDuration as V0RateLimitDurationProto,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    ConcurrencyLimitStrategy as ConcurrencyLimitStrategyProto,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    RateLimitDuration as RateLimitDurationProto,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    StickyStrategy as StickyStrategyProto,
)
from hatchet_sdk.runnables.types import StickyStrategy
from hatchet_sdk.types.concurrency import ConcurrencyStrategy
from hatchet_sdk.types.rate_limit import RateLimitDuration
from hatchet_sdk.utils.proto_enums import convert_python_literal_to_proto

CONCURRENCY_STRATEGY_LITERAL = ConcurrencyStrategy.model_fields["strategy"].annotation

# Values present in the proto enum but intentionally left out of the Literal,
# e.g. because the engine no longer honors them (see `// deprecated` in the
# .proto source). Keep this in sync with api-contracts/v1/workflows.proto.
# `test_deprecated_allowlist_is_current` fails if an entry here goes stale.
DEPRECATED_PROTO_VALUES: dict[str, set[str]] = {
    "ConcurrencyStrategy.strategy": {"DROP_NEWEST", "QUEUE_NEWEST"},
}

# Every Literal-backed proto enum in the SDK must be registered here, paired
# with each proto enum it is converted against at runtime. RateLimitDuration
# appears twice because `put_rate_limit` builds a v0 request while `RateLimit`
# builds a v1 one, and the same Literal must stay valid for both.
LITERAL_TO_PROTO_ENUM: list[tuple[str, Any, Any]] = [
    (
        "ConcurrencyStrategy.strategy",
        CONCURRENCY_STRATEGY_LITERAL,
        ConcurrencyLimitStrategyProto,
    ),
    ("RateLimitDuration", RateLimitDuration, RateLimitDurationProto),
    ("RateLimitDuration (v0)", RateLimitDuration, V0RateLimitDurationProto),
    ("StickyStrategy", StickyStrategy, StickyStrategyProto),
]

PARAM_IDS = [p[0] for p in LITERAL_TO_PROTO_ENUM]


def literal_string_values(name: str, literal_type: Any) -> set[str]:
    values = get_args(literal_type)

    assert values, (
        f"{name} did not resolve to a Literal type (got {literal_type!r}). "
        "Without its values, none of the checks in this file can run."
    )
    assert all(isinstance(v, str) for v in values), (
        f"{name} resolved to a Literal with non-string values: {values!r}. "
        "It must be a bare Literal of proto enum value names."
    )

    return set(values)


@pytest.mark.parametrize(
    "name,literal_type,proto_enum", LITERAL_TO_PROTO_ENUM, ids=PARAM_IDS
)
def test_literal_covers_every_proto_enum_value(
    name: str, literal_type: Any, proto_enum: Any
) -> None:
    literal_values = literal_string_values(name, literal_type)
    proto_values = {item.name for item in proto_enum.DESCRIPTOR.values}
    proto_values -= DEPRECATED_PROTO_VALUES.get(name, set())

    missing_from_literal = proto_values - literal_values
    assert not missing_from_literal, (
        f"{name} is missing proto values: {sorted(missing_from_literal)}. "
        "The engine supports these but the SDK can't express them."
    )

    extra_in_literal = literal_values - proto_values
    assert not extra_in_literal, (
        f"{name} has values that don't exist in the proto (or are marked "
        f"deprecated there): {sorted(extra_in_literal)}. These will fail at "
        "runtime when converted."
    )


@pytest.mark.parametrize(
    "name,literal_type,proto_enum", LITERAL_TO_PROTO_ENUM, ids=PARAM_IDS
)
def test_every_literal_value_converts_to_the_matching_proto_value(
    name: str, literal_type: Any, proto_enum: Any
) -> None:
    for value in sorted(literal_string_values(name, literal_type)):
        assert convert_python_literal_to_proto(value, proto_enum) == proto_enum.Value(
            value
        )


@pytest.mark.parametrize(
    "name,literal_type,proto_enum", LITERAL_TO_PROTO_ENUM, ids=PARAM_IDS
)
def test_none_and_invalid_values_are_handled(
    name: str, literal_type: Any, proto_enum: Any
) -> None:
    assert convert_python_literal_to_proto(None, proto_enum) is None

    with pytest.raises(ValueError):
        convert_python_literal_to_proto("NOT_A_REAL_VALUE", proto_enum)


def test_deprecated_allowlist_is_current() -> None:
    registered = {
        name: (literal, proto) for name, literal, proto in LITERAL_TO_PROTO_ENUM
    }

    for name, deprecated_values in DEPRECATED_PROTO_VALUES.items():
        assert name in registered, (
            f"DEPRECATED_PROTO_VALUES has an entry for {name!r}, which is not "
            "registered in LITERAL_TO_PROTO_ENUM. Fix the key or delete the entry."
        )

        literal_type, proto_enum = registered[name]
        proto_values = {item.name for item in proto_enum.DESCRIPTOR.values}

        gone_from_proto = deprecated_values - proto_values
        assert not gone_from_proto, (
            f"DEPRECATED_PROTO_VALUES[{name!r}] lists {sorted(gone_from_proto)}, "
            "which no longer exist in the proto enum. Delete the stale entries."
        )

        in_literal = deprecated_values & literal_string_values(name, literal_type)
        assert not in_literal, (
            f"{sorted(in_literal)} are marked deprecated but present in the "
            f"{name} Literal. Either un-deprecate them or remove them from the "
            "Literal."
        )
